# vsp Tool Reference

Complete documentation for all 96 MCP tools available in vsp.

**Mode Legend:**
- **Focused** - Available in focused mode (48 tools, default)
- **Expert** - Only available in expert mode (96 tools total)

---

## Unified Tools (2 tools) - Focused Mode

These tools replace 11 granular read/write operations with intelligent parameter-based routing:

| Tool | Description | Mode |
|------|-------------|------|
| `GetSource` | Unified read for any ABAP source. Parameters: `type` (PROG/CLAS/INTF/FUNC/FUGR/INCL/DDLS/VIEW/BDEF/SRVD/SRVB/MSAG), `name`, optional `parent` (for FUNC), optional `include` (for CLAS). | Focused |
| `WriteSource` | Unified write with auto-upsert. Parameters: `type` (PROG/CLAS/INTF/DDLS/BDEF/SRVD), `name`, `source`, `mode`, `options`. Supports create and update for classic ABAP and RAP types. | Focused |

**Benefits:** 70% token reduction, simplified tool selection, extensible for new types.

**RAP Support (NEW):** WriteSource now supports creating and updating CDS views (DDLS), behavior definitions (BDEF), and service definitions (SRVD).

---

## Search & Grep Tools (4 tools)

| Tool | Description | Mode |
|------|-------------|------|
| `SearchObject` | Quick search for ABAP objects by name pattern | Focused |
| `GrepObjects` | Regex search across multiple objects (array of URLs) | Focused |
| `GrepPackages` | Regex search across packages with recursive subpackage support | Focused |
| `GrepObject` | Regex search in single object | Expert |
| `GrepPackage` | Regex search in single package | Expert |

**Grep Features:**
- Full regex support (Go regexp syntax)
- Case-sensitive or case-insensitive matching
- Context lines (like `grep -C`)
- Object type filtering
- Max results limit

---

## Read Operations (15 tools)

| Tool | Description | Mode |
|------|-------------|------|
| `GetProgram` | Get ABAP program source | Expert |
| `GetClass` | Get ABAP class source | Expert |
| `GetInterface` | Get ABAP interface source | Expert |
| `GetFunction` | Get function module source | Expert |
| `GetFunctionGroup` | Get function group structure | Focused |
| `GetInclude` | Get ABAP include source | Expert |
| `GetTable` | Get table structure definition | Focused |
| `GetTableContents` | Get table data (supports SQL filtering) | Focused |
| `GetStructure` | Get structure definition | Expert |
| `GetPackage` | Get package contents | Focused |
| `GetTransaction` | Get transaction details | Expert |
| `GetTypeInfo` | Get data type information | Expert |
| `GetCDSDependencies` | Get CDS view dependency tree | Focused |
| `RunQuery` | Execute freestyle SQL query | Focused |

---

## System Information (2 tools) - NEW

| Tool | Description | Mode |
|------|-------------|------|
| `GetSystemInfo` | Get SAP system information (SID, release, kernel, database) | Focused |
| `GetInstalledComponents` | List installed software components with versions | Focused |

---

## Code Analysis (7 tools) - NEW

| Tool | Description | Mode |
|------|-------------|------|
| `GetCallGraph` | Get call hierarchy (callers/callees) for methods/functions | Focused |
| `GetObjectStructure` | Get object explorer tree structure | Focused |
| `GetCallersOf` | Get who calls this object (static call graph - up traversal) | Expert |
| `GetCalleesOf` | Get what this object calls (static call graph - down traversal) | Expert |
| `AnalyzeCallGraph` | Get statistics about call graph (nodes, edges, depth, types) | Expert |
| `CompareCallGraphs` | Compare static vs actual execution for test coverage analysis | Expert |
| `TraceExecution` | **COMPOSITE RCA TOOL**: Static graph + trace + comparison for root cause analysis | Expert |

---

## Development Tools (11 tools)

| Tool | Description | Mode |
|------|-------------|------|
| `SyntaxCheck` | Check source code for syntax errors | Focused |
| `Activate` | Activate an ABAP object | Expert |
| `ActivatePackage` | Batch activate all inactive objects in package | Focused |
| `RunUnitTests` | Execute ABAP Unit tests | Focused |
| `RunATCCheck` | Run ATC code quality checks | Focused |
| `CompareSource` | Unified diff between any two ABAP objects | Focused |
| `CloneObject` | Copy PROG/CLAS/INTF to new name | Focused |
| `GetClassInfo` | Quick class metadata (methods, attrs, interfaces) | Focused |
| `CreateTable` | Create DDIC table from JSON definition | Focused |
| `CreateStructure` | Create classic DDIC structure (SE11) from JSON definition | Focused |
| `CreatePackage` | Create local package ($...) | Focused |

### CreateStructure

Creates a classic DDIC structure (repository type `TABL`, ADT subtype `TABL/DS`) — **not** a database table. Workflow: create inactive structure → lock → write DDL source → unlock → activate (optional).

**Parameters:**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `name` | yes | Structure name (uppercase, max 30 chars). SAP rejects an underscore at the 2nd/3rd position (DT101): `ZS_TEST` invalid, `ZSTR_TEST` valid |
| `description` | yes | Short description |
| `package` | no | Target package (default: `$TMP`) |
| `components` | yes | JSON array; each entry has `name` plus **either** `data_element` **or** `type` (+`length`/`decimals`) |
| `transport` | no | Transport request (not needed for `$TMP`; requires `SAP_ALLOW_TRANSPORTABLE_EDITS`) |
| `activate_after_create` | no | Default `true`; if `false` the structure is left inactive |

**Example — local object ($TMP):**
```json
{
  "name": "ZSTR_VSP_TEST",
  "description": "VSP structure creation test",
  "package": "$TMP",
  "components": [
    {"name": "BUKRS", "data_element": "BUKRS"},
    {"name": "ANLN1", "data_element": "ANLN1"}
  ],
  "activate_after_create": true
}
```

**Example — transportable package:**
```json
{
  "name": "ZSTR_FI_ASSET",
  "description": "Asset key fields",
  "package": "ZFI",
  "transport": "NPLK900042",
  "components": [
    {"name": "BUKRS", "data_element": "BUKRS"},
    {"name": "ANLN1", "data_element": "ANLN1"}
  ],
  "activate_after_create": true
}
```

**Component types:**

A component is typed **either** by `data_element` (any existing DDIC data element) **or** by a predefined `type`. The two are mutually exclusive.

| Type | Needs | DDL |
|------|-------|-----|
| `CHAR`, `NUMC`, `RAW` | `length` (max 30000 / 255 / 32000) | `abap.char(10)` |
| `DEC` | `length` (1–31), `decimals` (0–14) | `abap.dec(15,2)` |
| `INT1`, `INT2`, `INT4`, `INT8`, `FLTP` | — | `abap.int4` |
| `DATS`, `TIMS`, `UTCLONG`, `CLNT`, `LANG` | — | `abap.dats` |
| `STRING`, `RAWSTRING` | — | `abap.string(0)` |

`CHAR32`/`NUMC10`/`RAW16` shorthand is accepted as an alternative to `type`+`length`.

```json
{
  "name": "ZSTR_DOC_ITEM",
  "description": "Document item",
  "package": "$TMP",
  "components": [
    {"name": "BUKRS",  "data_element": "BUKRS"},
    {"name": "AMOUNT", "type": "DEC", "length": 15, "decimals": 2},
    {"name": "NOTE",   "type": "CHAR", "length": 40},
    {"name": "SEQNO",  "type": "INT4"}
  ],
  "activate_after_create": true
}
```

**Limitations & behavior:**
- `CURR` and `QUAN` are **not** supported: DDIC requires a currency/unit reference annotation for them, which this API does not model yet. Use a data element (e.g. `WRBTR`, `MENGE`) instead.
- Unknown type names are rejected outright rather than being passed through as a data-element reference, so a typo fails fast with a clear message instead of an opaque activation error.
- Creation and activation status are reported separately; if activation fails, the structure exists but stays inactive and the activation messages are returned.
- All safety controls apply (read-only mode, allowed operations `C`/`L`/`A`, package restrictions, transport restrictions, transportable-edit opt-in).

---

## ATC (Code Quality) Tools (2 tools)

| Tool | Description | Mode |
|------|-------------|------|
| `RunATCCheck` | Run ATC check, returns findings with priority (1=Error, 2=Warning, 3=Info) | Focused |
| `GetATCCustomizing` | Get ATC system configuration | Expert |

**Example ATC Output:**
```json
{
  "summary": { "totalFindings": 3, "errors": 1, "warnings": 2 },
  "worklist": {
    "objects": [{
      "name": "ZCL_TEST",
      "findings": [{ "priority": 1, "checkTitle": "Syntax Check", "line": 42 }]
    }]
  }
}
```

---

## CRUD Operations (5 tools)

| Tool | Description | Mode |
|------|-------------|------|
| `LockObject` | Acquire edit lock | Focused |
| `UnlockObject` | Release edit lock | Focused |
| `CreateObject` | Create new object (program, class, interface, include, function group, function module, package, **DDLS, BDEF, SRVD, SRVB**) | Expert |
| `UpdateSource` | Write source code | Expert |
| `DeleteObject` | Delete an object | Expert |

**RAP Object Creation (NEW):** CreateObject now supports:
- `DDLS/DF` - CDS DDL Source (view definitions)
- `BDEF/BDO` - Behavior Definition
- `SRVD/SRV` - Service Definition
- `SRVB/SVB` - Service Binding (requires `service_definition`, optional `binding_version`, `binding_category`)

---

## Service Binding Operations (2 tools) - NEW

| Tool | Description | Mode |
|------|-------------|------|
| `PublishServiceBinding` | Publish a service binding to make it available as OData service | Expert |
| `UnpublishServiceBinding` | Unpublish a service binding | Expert |

**Parameters:**
- `service_name` (required) - Service binding name
- `service_version` (default: "0001")

---

## Class Include Operations (3 tools)

| Tool | Description | Mode |
|------|-------------|------|
| `GetClassInclude` | Get class include (definitions, implementations, macros, testclasses) | Expert |
| `CreateTestInclude` | Create test classes include | Expert |
| `UpdateClassInclude` | Update class include source | Expert |

---

## Workflow Tools (5 tools)

Composite operations that combine multiple ADT API calls:

| Tool | Description | Steps | Mode |
|------|-------------|-------|------|
| `EditSource` | **Surgical string replacement** (matches Edit tool pattern) | GetSource → FindReplace → SyntaxCheck → Lock → Update → Unlock → Activate | Focused |
| `WriteProgram` | Update program with activation | Lock → SyntaxCheck → Update → Unlock → Activate | Expert |
| `WriteClass` | Update class with activation | Lock → SyntaxCheck → Update → Unlock → Activate | Expert |
| `CreateAndActivateProgram` | Create new program | Create → UpdateSource → Activate | Expert |
| `CreateClassWithTests` | Create class with unit tests | Create → Lock → Update → CreateTestInclude → WriteTests → Unlock → Activate → RunUnitTests | Expert |

---

## File-Based Deployment Tools (5 tools)

Solves token limit problem for large files:

| Tool | Description | Mode |
|------|-------------|------|
| `ImportFromFile` | **File → SAP** - Smart deploy with auto create/update detection | Focused |
| `ExportToFile` | **SAP → File** - Save object source to local file | Focused |
| `DeployFromFile` | Legacy name for ImportFromFile | Expert |
| `SaveToFile` | Legacy name for ExportToFile | Expert |
| `RenameObject` | Rename object by creating copy | Expert |

**Supported Extensions:**
- `.clas.abap` - Classes
- `.prog.abap` - Programs
- `.intf.abap` - Interfaces
- `.fugr.abap` - Function Groups
- `.func.abap` - Function Modules
- `.ddls.asddls` - CDS DDL Sources (ABAPGit format)
- `.bdef.asbdef` - Behavior Definitions (ABAPGit format)
- `.srvd.srvdsrv` - Service Definitions (ABAPGit format)

---

## Code Intelligence Tools (7 tools)

| Tool | Description | Mode |
|------|-------------|------|
| `FindDefinition` | Navigate to symbol definition | Focused |
| `FindReferences` | Find all references to symbol | Focused |
| `CodeCompletion` | Get code completion suggestions | Expert |
| `PrettyPrint` | Format ABAP source code | Expert |
| `GetPrettyPrinterSettings` | Get formatter settings | Expert |
| `SetPrettyPrinterSettings` | Update formatter settings | Expert |
| `GetTypeHierarchy` | Get type hierarchy (supertypes/subtypes) | Expert |

---

## Transport Tools (3 tools)

| Tool | Description | Mode |
|------|-------------|------|
| `CreateTransport` | Create transport request | Expert |
| `GetTransportInfo` | Get transport details | Expert |
| `ReleaseTransport` | Release transport | Expert |
| `GetUserTransports` | List user's transports | Expert |
| `GetInactiveObjects` | List inactive objects | Expert |

---

## ExecuteABAP (Expert Mode)

Execute arbitrary ABAP code via unit test wrapper:

| Tool | Description | Mode |
|------|-------------|------|
| `ExecuteABAP` | Run ABAP code and capture output | Expert |

**Risk Levels:**
- `harmless` - Read-only, no external calls
- `dangerous` - Can write to DB, call external
- `critical` - Full system access

See [ExecuteABAP Implementation Report](reports/2025-12-05-004-execute-abap-implementation.md) for details.

---

## Runtime Errors / Short Dumps (2 tools) - RABAX

| Tool | Description | Mode |
|------|-------------|------|
| `GetDumps` | List runtime errors with filters (user, exception type, program, date range) | Focused |
| `GetDump` | Get full details of a specific dump including stack trace | Focused |

**Use Cases:**
- Monitor system health by checking recent dumps
- Debug production issues by examining dump details
- Track error patterns by exception type

---

## ABAP Profiler / Traces (2 tools) - ATRA

| Tool | Description | Mode |
|------|-------------|------|
| `ListTraces` | List ABAP runtime traces (profiler results) | Focused |
| `GetTrace` | Get trace analysis (hitlist, statements, dbAccesses) | Focused |

**Analysis Types:**
- `hitlist` - Hot spots by execution time
- `statements` - Statement-level trace
- `dbAccesses` - Database access analysis

---

## SQL Trace (2 tools) - ST05

| Tool | Description | Mode |
|------|-------------|------|
| `GetSQLTraceState` | Check if SQL trace is currently active | Focused |
| `ListSQLTraces` | List SQL trace files | Focused |

---

## Git / abapGit Tools (2 tools) - NEW v2.16.0

Exports ABAP objects using abapGit's native serialization. **Requires abapGit installed on SAP system.**

| Tool | Description | Mode |
|------|-------------|------|
| `GitTypes` | Get list of 158 supported abapGit object types | Focused |
| `GitExport` | Export packages/objects as abapGit-compatible ZIP (base64) | Focused |

**GitExport Parameters:**
- `packages` - Comma-separated package names (e.g., "$ZRAY,$TMP")
- `objects` - JSON array of objects: `[{"type":"CLAS","name":"ZCL_TEST"}]`
- `include_subpackages` - Include subpackages (default: true)

**Returns:** Base64-encoded ZIP with abapGit file structure:
```
src/
├── zcl_example.clas.abap      # Class source
├── zcl_example.clas.xml       # Class metadata
├── zif_example.intf.abap      # Interface source
└── ...
```

**Tool Group:** Git tools can be disabled with `--disabled-groups G`

**SAP Requirements:**
- `ZCL_ABAPGIT_OBJECTS` - Core serialization class
- `ZCL_ABAPGIT_FACTORY` - TADIR access factory
- Install via [abapGit standalone](https://github.com/abapGit/abapGit) or S/4HANA Developer Edition

---

## Install/Setup Tools (3 tools) - NEW v2.17.0

Deploy VSP components and dependencies to SAP systems via ADT.

| Tool | Description | Mode |
|------|-------------|------|
| `InstallZADTVSP` | Deploy ZADT_VSP WebSocket handler (6 ABAP objects) | Focused |
| `InstallAbapGit` | Deploy abapGit from embedded ZIP (standalone or dev edition) | Focused |
| `ListDependencies` | List available dependencies for installation | Focused |

**InstallZADTVSP Parameters:**
- `package` - Target package name (default: `$ZADT_VSP`)
- `skip_git_service` - Skip Git service if no abapGit (default: auto-detected)
- `check_only` - Only check prerequisites, don't deploy

**InstallAbapGit Parameters:**
- `edition` - `standalone` (single program) or `dev` (full packages)
- `package` - Target package (default: `$ABAPGIT` or `$ZGIT_DEV`)
- `check_only` - Only show deployment plan

**Architecture:**
```
embedded/
├── abap/           # ZADT_VSP source (raw ABAP, go:embed)
│   ├── zif_vsp_service.intf.abap
│   ├── zcl_vsp_*.clas.abap
│   └── embed.go
│
└── deps/           # Dependencies (abapGit ZIP format)
    ├── abapgit-standalone.zip  # Placeholder
    ├── abapgit-dev.zip         # Placeholder
    └── embed.go                # Unzip + deploy logic
```

**Tool Group:** Install tools can be disabled with `--disabled-groups I`

---

## Report Execution Tools (6 tools) - NEW v2.19.0

Execute ABAP reports with parameters and capture ALV output. Includes async pattern for long-running reports.

| Tool | Description | Mode |
|------|-------------|------|
| `RunReport` | Execute report with params/variant, capture ALV output | Focused |
| `RunReportAsync` | Start report in background, returns task_id | Focused |
| `GetAsyncResult` | Poll or wait for async task completion | Focused |
| `GetVariants` | List available variants for a report | Focused |
| `GetTextElements` | Get selection texts and text symbols | Focused |
| `SetTextElements` | Update selection texts and text symbols | Focused |

**Async Pattern:**
```
1. RunReportAsync(report="RFITEMGL", params={...})
   → {"task_id": "report_1736034567_1", "status": "started"}

2. GetAsyncResult(task_id="...", wait=true)
   → Blocks up to 60s, returns full result when complete
```

**Requires:** ZADT_VSP WebSocket handler deployed to SAP system.

**Tool Group:** Report tools can be disabled with `--disabled-groups R`

---

## Tool Count Summary

| Mode | Tools | Description |
|------|-------|-------------|
| **Focused** | 54 | Essential tools for AI-assisted development |
| **Expert** | 99 | All tools including low-level operations and RAP creation |

**Token Savings with Focused Mode:**
- Tool definitions: 50% reduction (~5,000 → ~2,500 tokens)
- Typical workflow: 60% reduction
- Decision clarity: 48 choices instead of 96
