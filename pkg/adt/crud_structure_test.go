package adt

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// structureMock is a test transport for the CreateStructure flow. Unlike
// methodPathMock it records the full request (query, headers, body) so
// tests can assert corrNr/lockHandle parameters, Content-Type/Accept
// headers, session type, and payload contents.
type structureMock struct {
	routes []routedResponse
	calls  []structureCall
}

type structureCall struct {
	method      string
	path        string
	query       url.Values
	contentType string
	accept      string
	sessionType string
	body        string
}

func (m *structureMock) Do(req *http.Request) (*http.Response, error) {
	body := ""
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		body = string(b)
	}
	m.calls = append(m.calls, structureCall{
		method:      req.Method,
		path:        req.URL.Path,
		query:       req.URL.Query(),
		contentType: req.Header.Get("Content-Type"),
		accept:      req.Header.Get("Accept"),
		sessionType: req.Header.Get("X-sap-adt-sessiontype"),
		body:        body,
	})
	for _, r := range m.routes {
		if r.method != "" && r.method != req.Method {
			continue
		}
		if r.pathSubstring == "" || strings.Contains(req.URL.Path, r.pathSubstring) {
			if r.err != nil {
				return nil, r.err
			}
			h := http.Header{}
			h.Set("X-CSRF-Token", "test-token")
			return &http.Response{
				StatusCode: r.status,
				Body:       io.NopCloser(strings.NewReader(r.body)),
				Header:     h,
			}, nil
		}
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader("not routed: " + req.Method + " " + req.URL.Path)),
		Header:     http.Header{},
	}, nil
}

// findCall returns the first recorded call matching method and path substring.
func (m *structureMock) findCall(method, pathSubstring string) *structureCall {
	for i := range m.calls {
		if m.calls[i].method == method && strings.Contains(m.calls[i].path, pathSubstring) {
			return &m.calls[i]
		}
	}
	return nil
}

// findCallWithQuery returns the first call matching method, path substring,
// and an exact query parameter value (used to tell LOCK from UNLOCK).
func (m *structureMock) findCallWithQuery(method, pathSubstring, key, value string) *structureCall {
	for i := range m.calls {
		c := &m.calls[i]
		if c.method == method && strings.Contains(c.path, pathSubstring) && c.query.Get(key) == value {
			return c
		}
	}
	return nil
}

func newStructureClient(t *testing.T, mock *structureMock, opts ...Option) *Client {
	t.Helper()
	cfg := NewConfig("https://sap.example.com:44300", "testuser", "testpass", opts...)
	transport := NewTransportWithClient(cfg, mock)
	return NewClientWithTransport(cfg, transport)
}

// structureHappyRoutes stages the full create → lock → PUT source →
// unlock → activate sequence. Order matters: more specific paths first,
// because routing is first-match on substring.
func structureHappyRoutes() []routedResponse {
	return []routedResponse{
		resp("", "discovery", 200, "ok"),
		resp(http.MethodPut, "/source/main", 200, ""),
		// LOCK and UNLOCK both POST to the object URL; the lock XML body is
		// ignored by the unlock path.
		resp(http.MethodPost, "/ddic/structures/zs_", 200, lockResponseXML),
		resp(http.MethodPost, "/activation", 200, ""),
		resp(http.MethodPost, "/ddic/structures", 201, ""),
	}
}

func validStructureOpts() CreateStructureOptions {
	return CreateStructureOptions{
		Name:        "ZS_VSP_TEST",
		Description: "VSP structure creation test",
		Package:     "$TMP",
		Components: []StructureComponent{
			{Name: "BUKRS", DataElement: "BUKRS"},
			{Name: "ANLN1", DataElement: "ANLN1"},
		},
		Activate: true,
	}
}

// --- Payload generation ---

func TestBuildStructureXML(t *testing.T) {
	opts, err := validateCreateStructureOptions(validStructureOpts())
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	xml := buildStructureXML(opts)

	for _, want := range []string{
		`adtcore:type="TABL/DS"`, // structure, NOT database table (TABL/DT)
		`adtcore:name="ZS_VSP_TEST"`,
		`adtcore:description="VSP structure creation test"`,
		`<adtcore:packageRef adtcore:name="$TMP"/>`,
		`xmlns:blue="http://www.sap.com/wbobj/blue"`,
		`xmlns:adtcore="http://www.sap.com/adt/core"`,
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("structure XML missing %q:\n%s", want, xml)
		}
	}
	if strings.Contains(xml, "TABL/DT") {
		t.Errorf("structure XML must not use database table type TABL/DT:\n%s", xml)
	}
}

func TestBuildStructureXML_EscapesDescription(t *testing.T) {
	opts := validStructureOpts()
	opts.Description = `Test <&"> structure`
	opts, err := validateCreateStructureOptions(opts)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	xml := buildStructureXML(opts)
	if !strings.Contains(xml, "Test &lt;&amp;&quot;&gt; structure") {
		t.Errorf("description not XML-escaped:\n%s", xml)
	}
}

func TestGenerateStructureDDL(t *testing.T) {
	opts, err := validateCreateStructureOptions(validStructureOpts())
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	ddl, err := generateStructureDDL(opts)
	if err != nil {
		t.Fatalf("unexpected DDL error: %v", err)
	}

	want := "@EndUserText.label : 'VSP structure creation test'\n" +
		"@AbapCatalog.enhancement.category : #NOT_EXTENSIBLE\n" +
		"define structure zs_vsp_test {\n" +
		"  bukrs : bukrs;\n" +
		"  anln1 : anln1;\n" +
		"}\n"
	if ddl != want {
		t.Errorf("DDL mismatch:\ngot:\n%s\nwant:\n%s", ddl, want)
	}
}

func TestGenerateStructureDDL_EscapesQuoteInLabel(t *testing.T) {
	opts := validStructureOpts()
	opts.Description = "it's a test"
	opts, err := validateCreateStructureOptions(opts)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	ddl, err := generateStructureDDL(opts)
	if err != nil {
		t.Fatalf("unexpected DDL error: %v", err)
	}
	if !strings.Contains(ddl, "@EndUserText.label : 'it''s a test'") {
		t.Errorf("label quote not escaped:\n%s", ddl)
	}
}

// --- Predefined component types ---

// TestStructureComponentDDLType covers the predefined-type renderings that
// were round-tripped against a real SAP system (create → activate → DD03L).
func TestStructureComponentDDLType(t *testing.T) {
	cases := []struct {
		comp StructureComponent
		want string
	}{
		{StructureComponent{Name: "F", DataElement: "BUKRS"}, "bukrs"},
		{StructureComponent{Name: "F", Type: "CHAR", Length: 10}, "abap.char(10)"},
		{StructureComponent{Name: "F", Type: "NUMC", Length: 8}, "abap.numc(8)"},
		{StructureComponent{Name: "F", Type: "RAW", Length: 16}, "abap.raw(16)"},
		{StructureComponent{Name: "F", Type: "DEC", Length: 15, Decimals: 2}, "abap.dec(15,2)"},
		{StructureComponent{Name: "F", Type: "DEC", Length: 15}, "abap.dec(15,0)"},
		{StructureComponent{Name: "F", Type: "INT1"}, "abap.int1"},
		{StructureComponent{Name: "F", Type: "INT2"}, "abap.int2"},
		{StructureComponent{Name: "F", Type: "INT4"}, "abap.int4"},
		{StructureComponent{Name: "F", Type: "INT8"}, "abap.int8"},
		{StructureComponent{Name: "F", Type: "FLTP"}, "abap.fltp"},
		{StructureComponent{Name: "F", Type: "DATS"}, "abap.dats"},
		{StructureComponent{Name: "F", Type: "TIMS"}, "abap.tims"},
		{StructureComponent{Name: "F", Type: "UTCLONG"}, "abap.utclong"},
		{StructureComponent{Name: "F", Type: "CLNT"}, "abap.clnt"},
		{StructureComponent{Name: "F", Type: "LANG"}, "abap.lang"},
		{StructureComponent{Name: "F", Type: "STRING"}, "abap.string(0)"},
		{StructureComponent{Name: "F", Type: "RAWSTRING"}, "abap.rawstring(0)"},
		// CHARnn / NUMCnn / RAWnn shorthand, as CreateTable accepts.
		{StructureComponent{Name: "F", Type: "CHAR32"}, "abap.char(32)"},
		{StructureComponent{Name: "F", Type: "NUMC10"}, "abap.numc(10)"},
		{StructureComponent{Name: "F", Type: "RAW16"}, "abap.raw(16)"},
	}

	for _, tc := range cases {
		label := tc.comp.Type
		if label == "" {
			label = "data_element:" + tc.comp.DataElement
		}
		t.Run(label, func(t *testing.T) {
			got, err := structureComponentDDLType(tc.comp)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestStructureComponentDDLType_Rejects guards the design decision that an
// unrecognized type is an error, never a silent data-element reference.
func TestStructureComponentDDLType_Rejects(t *testing.T) {
	cases := []struct {
		name    string
		comp    StructureComponent
		wantErr string
	}{
		{"unknown type", StructureComponent{Type: "CHAR32X"}, "unknown type"},
		{"typo not treated as data element", StructureComponent{Type: "CHARR", Length: 10}, "unknown type"},
		{"char without length", StructureComponent{Type: "CHAR"}, "requires length between 1 and 30000"},
		{"char length too large", StructureComponent{Type: "CHAR", Length: 30001}, "requires length between 1 and 30000"},
		{"numc length too large", StructureComponent{Type: "NUMC", Length: 256}, "requires length between 1 and 255"},
		{"char with decimals", StructureComponent{Type: "CHAR", Length: 10, Decimals: 2}, "does not take decimals"},
		{"int4 with length", StructureComponent{Type: "INT4", Length: 4}, "does not take a length"},
		{"int4 with decimals", StructureComponent{Type: "INT4", Decimals: 2}, "does not take decimals"},
		{"dec length too large", StructureComponent{Type: "DEC", Length: 32, Decimals: 2}, "length between 1 and 31"},
		{"dec decimals too large", StructureComponent{Type: "DEC", Length: 20, Decimals: 15}, "decimals between 0 and 14"},
		{"dec decimals exceed length", StructureComponent{Type: "DEC", Length: 3, Decimals: 5}, "cannot exceed length"},
		{"shorthand conflicts with length", StructureComponent{Type: "CHAR32", Length: 10}, "conflicting"},
		// CURR/QUAN need a currency/unit reference annotation — verified
		// against a real system, where activation fails without it.
		{"curr unsupported", StructureComponent{Type: "CURR", Length: 15, Decimals: 2}, "requires a currency/unit reference"},
		{"quan unsupported", StructureComponent{Type: "QUAN", Length: 13, Decimals: 3}, "requires a currency/unit reference"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := structureComponentDDLType(tc.comp)
			if err == nil {
				t.Fatalf("expected error containing %q, got success", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestGenerateStructureDDL_MixedComponentTypes(t *testing.T) {
	opts := CreateStructureOptions{
		Name:        "ZSTR_MIX",
		Description: "Mixed component types",
		Package:     "$TMP",
		Components: []StructureComponent{
			{Name: "BUKRS", DataElement: "BUKRS"},
			{Name: "F_CHAR", Type: "char", Length: 10},
			{Name: "F_DEC", Type: "DEC", Length: 15, Decimals: 2},
			{Name: "F_INT4", Type: "INT4"},
		},
	}
	opts, err := validateCreateStructureOptions(opts)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	ddl, err := generateStructureDDL(opts)
	if err != nil {
		t.Fatalf("unexpected DDL error: %v", err)
	}

	want := "@EndUserText.label : 'Mixed component types'\n" +
		"@AbapCatalog.enhancement.category : #NOT_EXTENSIBLE\n" +
		"define structure zstr_mix {\n" +
		"  bukrs : bukrs;\n" +
		"  f_char : abap.char(10);\n" +
		"  f_dec : abap.dec(15,2);\n" +
		"  f_int4 : abap.int4;\n" +
		"}\n"
	if ddl != want {
		t.Errorf("DDL mismatch:\ngot:\n%s\nwant:\n%s", ddl, want)
	}
}

// --- Validation ---

func TestCreateStructure_ValidationErrors(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*CreateStructureOptions)
		wantErr string
	}{
		{"empty name", func(o *CreateStructureOptions) { o.Name = "" }, "structure name"},
		{"name too long", func(o *CreateStructureOptions) { o.Name = strings.Repeat("Z", 31) }, "structure name"},
		{"invalid structure name", func(o *CreateStructureOptions) { o.Name = "ZS TEST!" }, "invalid structure name"},
		{"name starting with digit", func(o *CreateStructureOptions) { o.Name = "1ZS_TEST" }, "invalid structure name"},
		{"missing description", func(o *CreateStructureOptions) { o.Description = "  " }, "description is required"},
		{"empty components", func(o *CreateStructureOptions) { o.Components = nil }, "at least one component"},
		{"missing component name", func(o *CreateStructureOptions) { o.Components[0].Name = "" }, "name is required"},
		{"invalid component name", func(o *CreateStructureOptions) { o.Components[0].Name = "FIELD-1" }, "invalid component name"},
		{"namespaced component name", func(o *CreateStructureOptions) { o.Components[0].Name = "/NS/FIELD" }, "invalid component name"},
		{"missing type and data element", func(o *CreateStructureOptions) { o.Components[0].DataElement = "" }, "either data_element or type is required"},
		{"data element and type both set", func(o *CreateStructureOptions) { o.Components[0].Type = "CHAR" }, "mutually exclusive"},
		{"invalid data element", func(o *CreateStructureOptions) { o.Components[0].DataElement = "BU KRS" }, "invalid data element"},
		{"unknown predefined type", func(o *CreateStructureOptions) {
			o.Components[0].DataElement = ""
			o.Components[0].Type = "CHAR32X"
		}, "unknown type"},
		{"predefined type missing length", func(o *CreateStructureOptions) {
			o.Components[0].DataElement = ""
			o.Components[0].Type = "CHAR"
		}, "requires length"},
		{"duplicate component", func(o *CreateStructureOptions) {
			o.Components = append(o.Components, StructureComponent{Name: "bukrs", DataElement: "BUKRS"})
		}, "duplicate component"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &structureMock{routes: structureHappyRoutes()}
			client := newStructureClient(t, mock)

			opts := validStructureOpts()
			tc.mutate(&opts)

			result, err := client.CreateStructure(context.Background(), opts)
			if err == nil {
				t.Fatalf("expected validation error containing %q, got success", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
			if result != nil {
				t.Errorf("expected nil result on validation failure, got %+v", result)
			}
			if len(mock.calls) != 0 {
				t.Errorf("validation failure must not reach SAP, but %d HTTP calls were made", len(mock.calls))
			}
		})
	}
}

func TestCreateStructure_NamespacedNameAccepted(t *testing.T) {
	opts := validStructureOpts()
	opts.Name = "/ABC/S_TEST"
	if _, err := validateCreateStructureOptions(opts); err != nil {
		t.Errorf("namespaced structure name should be accepted: %v", err)
	}
}

// --- Request sequence ---

func TestCreateStructure_HappyPathTmp(t *testing.T) {
	mock := &structureMock{routes: structureHappyRoutes()}
	client := newStructureClient(t, mock)

	result, err := client.CreateStructure(context.Background(), validStructureOpts())
	if err != nil {
		t.Fatalf("CreateStructure failed: %v", err)
	}
	if !result.Created || !result.Activated || result.Status != "active" {
		t.Errorf("unexpected result: %+v", result)
	}

	// 1. Creation POST: endpoint, method, headers, subtype, package, no corrNr
	create := mock.findCall(http.MethodPost, "/sap/bc/adt/ddic/structures")
	if create == nil || create.path != "/sap/bc/adt/ddic/structures" {
		t.Fatalf("creation POST to /sap/bc/adt/ddic/structures not found; calls: %+v", mock.calls)
	}
	if create.contentType != "application/*" {
		t.Errorf("creation Content-Type = %q, want application/*", create.contentType)
	}
	if !strings.Contains(create.body, `adtcore:type="TABL/DS"`) {
		t.Errorf("creation body missing TABL/DS subtype:\n%s", create.body)
	}
	if !strings.Contains(create.body, `<adtcore:packageRef adtcore:name="$TMP"/>`) {
		t.Errorf("creation body missing package ref:\n%s", create.body)
	}
	if create.query.Get("corrNr") != "" {
		t.Errorf("$TMP creation must not send corrNr, got %q", create.query.Get("corrNr"))
	}

	// 2. Lock on object URL
	lock := mock.findCallWithQuery(http.MethodPost, "/ddic/structures/zs_vsp_test", "_action", "LOCK")
	if lock == nil {
		t.Fatalf("lock POST not found; calls: %+v", mock.calls)
	}

	// 3. Source PUT with lock handle, stateful, text/plain
	put := mock.findCall(http.MethodPut, "/ddic/structures/zs_vsp_test/source/main")
	if put == nil {
		t.Fatalf("source PUT not found; calls: %+v", mock.calls)
	}
	if put.query.Get("lockHandle") != "TESTHANDLE" {
		t.Errorf("source PUT lockHandle = %q, want TESTHANDLE", put.query.Get("lockHandle"))
	}
	if put.query.Get("corrNr") != "" {
		t.Errorf("$TMP source PUT must not send corrNr, got %q", put.query.Get("corrNr"))
	}
	if put.contentType != "text/plain; charset=utf-8" {
		t.Errorf("source PUT Content-Type = %q, want text/plain; charset=utf-8", put.contentType)
	}
	if put.sessionType != "stateful" {
		t.Errorf("source PUT session type = %q, want stateful (must share the lock's session)", put.sessionType)
	}
	if !strings.Contains(put.body, "define structure zs_vsp_test") {
		t.Errorf("source PUT body is not structure DDL:\n%s", put.body)
	}
	if !strings.Contains(put.body, "bukrs : bukrs;") || !strings.Contains(put.body, "anln1 : anln1;") {
		t.Errorf("source PUT body missing components:\n%s", put.body)
	}

	// 4. Unlock before activation
	unlock := mock.findCallWithQuery(http.MethodPost, "/ddic/structures/zs_vsp_test", "_action", "UNLOCK")
	if unlock == nil {
		t.Fatalf("unlock POST not found; calls: %+v", mock.calls)
	}

	// 5. Activation with the structure's object reference
	activate := mock.findCall(http.MethodPost, "/sap/bc/adt/activation")
	if activate == nil {
		t.Fatalf("activation POST not found; calls: %+v", mock.calls)
	}
	if !strings.Contains(activate.body, `adtcore:uri="/sap/bc/adt/ddic/structures/zs_vsp_test"`) ||
		!strings.Contains(activate.body, `adtcore:name="ZS_VSP_TEST"`) {
		t.Errorf("activation body missing structure reference:\n%s", activate.body)
	}
}

func TestCreateStructure_WithTransport(t *testing.T) {
	mock := &structureMock{routes: structureHappyRoutes()}
	client := newStructureClient(t, mock, WithAllowTransportableEdits())

	opts := validStructureOpts()
	opts.Package = "ZFI"
	opts.Transport = "NPLK900042"

	result, err := client.CreateStructure(context.Background(), opts)
	if err != nil {
		t.Fatalf("CreateStructure failed: %v", err)
	}
	if result.Transport != "NPLK900042" || result.Package != "ZFI" {
		t.Errorf("unexpected result: %+v", result)
	}

	create := mock.findCall(http.MethodPost, "/sap/bc/adt/ddic/structures")
	if create == nil {
		t.Fatal("creation POST not found")
	}
	if create.query.Get("corrNr") != "NPLK900042" {
		t.Errorf("creation corrNr = %q, want NPLK900042", create.query.Get("corrNr"))
	}
	if !strings.Contains(create.body, `<adtcore:packageRef adtcore:name="ZFI"/>`) {
		t.Errorf("creation body missing package ZFI:\n%s", create.body)
	}

	put := mock.findCall(http.MethodPut, "/source/main")
	if put == nil {
		t.Fatal("source PUT not found")
	}
	if put.query.Get("corrNr") != "NPLK900042" {
		t.Errorf("source PUT corrNr = %q, want NPLK900042", put.query.Get("corrNr"))
	}
}

func TestCreateStructure_TransportBlockedWithoutOptIn(t *testing.T) {
	// Consistent with CreateObject/UpdateSource: a transport requires
	// AllowTransportableEdits.
	mock := &structureMock{routes: structureHappyRoutes()}
	client := newStructureClient(t, mock) // no WithAllowTransportableEdits

	opts := validStructureOpts()
	opts.Package = "ZFI"
	opts.Transport = "NPLK900042"

	_, err := client.CreateStructure(context.Background(), opts)
	if err == nil {
		t.Fatal("expected transportable-edit block, got success")
	}
	if !strings.Contains(err.Error(), "transportable") {
		t.Errorf("unexpected error: %v", err)
	}
	if len(mock.calls) != 0 {
		t.Errorf("blocked operation must not reach SAP, but %d HTTP calls were made", len(mock.calls))
	}
}

// --- Error paths ---

func TestCreateStructure_CreateErrorSurfaced(t *testing.T) {
	sapError := `<?xml version="1.0" encoding="utf-8"?><exc:exception xmlns:exc="http://www.sap.com/abapxml/types/communicationframework"><message lang="EN">Data element ZDE_MISSING does not exist</message></exc:exception>`
	mock := &structureMock{
		routes: []routedResponse{
			resp("", "discovery", 200, "ok"),
			resp(http.MethodPost, "/ddic/structures", 400, sapError),
		},
	}
	client := newStructureClient(t, mock)

	result, err := client.CreateStructure(context.Background(), validStructureOpts())
	if err == nil {
		t.Fatal("expected creation error, got success")
	}
	if !strings.Contains(err.Error(), "creating structure object") {
		t.Errorf("error not attributed to creation step: %v", err)
	}
	if !strings.Contains(err.Error(), "Data element ZDE_MISSING does not exist") {
		t.Errorf("SAP error detail lost: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result when creation fails, got %+v", result)
	}
}

func TestCreateStructure_AlreadyExists(t *testing.T) {
	sapError := `<?xml version="1.0" encoding="utf-8"?><exc:exception xmlns:exc="http://www.sap.com/abapxml/types/communicationframework"><type id="ExceptionResourceAlreadyExists"/><message lang="EN">Resource Structure ZS_VSP_TEST does already exist</message></exc:exception>`
	mock := &structureMock{
		routes: []routedResponse{
			resp("", "discovery", 200, "ok"),
			resp(http.MethodPost, "/ddic/structures", 405, sapError),
		},
	}
	client := newStructureClient(t, mock)

	_, err := client.CreateStructure(context.Background(), validStructureOpts())
	if err == nil {
		t.Fatal("expected already-exists error, got success")
	}
	if !strings.Contains(err.Error(), "does already exist") {
		t.Errorf("already-exists detail lost: %v", err)
	}
}

func TestCreateStructure_SourcePutFailureUnlocks(t *testing.T) {
	mock := &structureMock{
		routes: []routedResponse{
			resp("", "discovery", 200, "ok"),
			resp(http.MethodPut, "/source/main", 500, "syntax error in DDL"),
			resp(http.MethodPost, "/ddic/structures/zs_", 200, lockResponseXML),
			resp(http.MethodPost, "/ddic/structures", 201, ""),
		},
	}
	client := newStructureClient(t, mock)

	result, err := client.CreateStructure(context.Background(), validStructureOpts())
	if err == nil {
		t.Fatal("expected source PUT error, got success")
	}
	if !strings.Contains(err.Error(), "setting structure source") {
		t.Errorf("error not attributed to source step: %v", err)
	}
	if result == nil || !result.Created || result.Activated {
		t.Errorf("expected created-but-not-activated result, got %+v", result)
	}
	if mock.findCallWithQuery(http.MethodPost, "/ddic/structures/zs_vsp_test", "_action", "UNLOCK") == nil {
		t.Errorf("lock must be released after source PUT failure; calls: %+v", mock.calls)
	}
}

func TestCreateStructure_ActivationFails(t *testing.T) {
	activationErrorXML := `<?xml version="1.0" encoding="UTF-8"?>
<chkl:messages xmlns:chkl="http://www.sap.com/abapxml/checklist">
  <msg objDescr="Structure ZS_VSP_TEST" type="E" line="1" href="/sap/bc/adt/ddic/structures/zs_vsp_test" forceSupported="false">
    <shortText><txt>Field BUKRS: Component type or domain used not active or does not exist</txt></shortText>
  </msg>
</chkl:messages>`
	routes := []routedResponse{
		resp("", "discovery", 200, "ok"),
		resp(http.MethodPut, "/source/main", 200, ""),
		resp(http.MethodPost, "/ddic/structures/zs_", 200, lockResponseXML),
		resp(http.MethodPost, "/activation", 200, activationErrorXML),
		resp(http.MethodPost, "/ddic/structures", 201, ""),
	}
	mock := &structureMock{routes: routes}
	client := newStructureClient(t, mock)

	result, err := client.CreateStructure(context.Background(), validStructureOpts())
	if err == nil {
		t.Fatal("activation failure must be reported as an error")
	}
	if !strings.Contains(err.Error(), "activating structure") {
		t.Errorf("error not attributed to activation step: %v", err)
	}
	if result == nil {
		t.Fatal("expected partial result when activation fails")
	}
	if !result.Created {
		t.Errorf("creation succeeded and must be reported: %+v", result)
	}
	if result.Activated || result.Status != "inactive" {
		t.Errorf("activation failed but result claims otherwise: %+v", result)
	}
	if len(result.ActivationMessages) == 0 || !strings.Contains(result.ActivationMessages[0], "not active or does not exist") {
		t.Errorf("activation messages missing: %+v", result.ActivationMessages)
	}
}

func TestCreateStructure_NoActivation(t *testing.T) {
	mock := &structureMock{routes: structureHappyRoutes()}
	client := newStructureClient(t, mock)

	opts := validStructureOpts()
	opts.Activate = false

	result, err := client.CreateStructure(context.Background(), opts)
	if err != nil {
		t.Fatalf("CreateStructure failed: %v", err)
	}
	if !result.Created || result.Activated || result.Status != "inactive" {
		t.Errorf("expected created-inactive result, got %+v", result)
	}
	if mock.findCall(http.MethodPost, "/sap/bc/adt/activation") != nil {
		t.Errorf("activation must not be called when Activate=false; calls: %+v", mock.calls)
	}
}

// --- Safety controls ---

func TestCreateStructure_ReadOnlyBlocked(t *testing.T) {
	mock := &structureMock{routes: structureHappyRoutes()}
	client := newStructureClient(t, mock, WithReadOnly())

	_, err := client.CreateStructure(context.Background(), validStructureOpts())
	if err == nil {
		t.Fatal("expected read-only block, got success")
	}
	if len(mock.calls) != 0 {
		t.Errorf("read-only block must not reach SAP, but %d HTTP calls were made", len(mock.calls))
	}
}

func TestCreateStructure_PackageNotAllowed(t *testing.T) {
	mock := &structureMock{routes: structureHappyRoutes()}
	client := newStructureClient(t, mock, WithAllowedPackages("ZOK"))

	opts := validStructureOpts()
	opts.Package = "ZFORBIDDEN"

	_, err := client.CreateStructure(context.Background(), opts)
	if err == nil {
		t.Fatal("expected package restriction block, got success")
	}
	if !strings.Contains(err.Error(), "ZFORBIDDEN") {
		t.Errorf("unexpected error: %v", err)
	}
	if len(mock.calls) != 0 {
		t.Errorf("package block must not reach SAP, but %d HTTP calls were made", len(mock.calls))
	}
}
