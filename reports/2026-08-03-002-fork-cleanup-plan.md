# Fork-Strategie: Umsetzungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Die in [`2026-08-03-001-fork-strategy.md`](2026-08-03-001-fork-strategy.md) beschlossene Fork-Strategie in Kraft setzen: Dokumentation verankern, ein Test-Gate schaffen, und die vier offenen Altlasten aus Abschnitt 9 auflösen.

**Architecture:** Sieben unabhängig prüfbare Aufgaben in fester Reihenfolge. Zuerst die Dokumentation (risikofrei, legt die Regeln fest), dann die Werkzeuge (Go-Toolchain, CI), dann die Aufräumarbeiten von der harmlosesten zur inhaltlich schwierigsten. Keine Aufgabe schreibt Historie um, die bereits in `main` oder in einem offenen PR steckt.

**Tech Stack:** Git, GitHub CLI (`gh`), Go 1.25, GitHub Actions.

## Global Constraints

- **Keine KI-Attribution in Commits.** Keine Co-Author-Trailer für Agenten oder Assistenten, keine „Generated with"-Footer, kein Hinweis auf Werkzeugbeteiligung in Subject, Body oder Trailer — auch nicht in Commit-Messages, die aus bestehenden Branches übernommen werden. Gilt ebenso für PR-Titel und -Beschreibungen, Issues, Branchnamen und Tags.
- **Sanitize-Policy** (`CLAUDE.md`): keine realen SAP-Benutzer, Hostnamen, IPs, Transportnummern oder Systemaliase, die eine Live-Box benennen. Ersatz: `TESTUSER`, `devsys`, `dev.example.local`, `TR-EXAMPLE`.
  - **Bekannter Fehlalarm:** `pkg/adt/crud_reconcile_test.go` enthält in der Konstanten `lockResponseXML` ein `CORRNR`-Element mit einem transportförmigen Platzhalter. Er trifft das strukturelle Muster `[A-Z][0-9]{2}K[0-9]{6}`, ist aber eine synthetische Fixture, die schon in `main` **und** upstream steht. Task 5 und 6 fassen diese Datei an — der Scan darf dort nicht als Leak gewertet werden. Der Wert wird hier bewusst nicht zitiert, sonst schlägt der Scan auf diesem Plandokument selbst an.
  - Fremde Commit-Messages aus Upstream-PRs werden **nicht** umgeschrieben. Die Identifier darin gehören ihren Autoren und sind bereits öffentlich; ein Rewrite würde die Attribution zerstören. Die Policy zielt auf eigene Leaks.
- **Regel 1:** Upstream-taugliche Arbeit zweigt von `upstream/main` ab, nie von `main`.
- **Regel 2:** Nie cherry-picken, immer mergen. Einzige Ausnahme: nachträgliches Upstreamen (`FORK.md`).
- **Immer `git merge --no-ff`**, nie „Squash and merge".
- **Branches mit offenem Upstream-PR niemals löschen:** `feat/incl-write-support`, `fix/csrf-head-fallback-and-session-type`, `fix/search-type-filter-issue-119`.
- Tags werden ausschließlich von `main` gezogen, und nur bei grünem Build und Test.

---

### Task 1: Fork-Dokumentation committen

Risikofrei, braucht keine Go-Toolchain, und legt die Regeln fest, nach denen alle folgenden Aufgaben laufen. Deshalb zuerst.

**Files:**
- Create: `FORK.md` *(bereits im Working Tree)*
- Create: `reports/2026-08-03-001-fork-strategy.md` *(bereits im Working Tree)*
- Create: `reports/2026-08-03-002-fork-cleanup-plan.md` *(diese Datei)*
- Modify: `CLAUDE.md` *(Doc-intent-Zeile + Fork-Hinweisblock, bereits im Working Tree)*
- Modify: `README.md` *(Fork-Notice im Kopf-Blockquote, bereits im Working Tree)*

**Interfaces:**
- Produces: `FORK.md` als operative Referenz, auf die Task 5, 6 und 7 ihre Entscheidungstabellen schreiben.

- [ ] **Step 1: Working Tree prüfen**

```bash
git status --short
```

Erwartet: genau fünf Einträge — `?? FORK.md`, `?? reports/2026-08-03-001-fork-strategy.md`, `?? reports/2026-08-03-002-fork-cleanup-plan.md`, `M CLAUDE.md`, `M README.md`. Weicht das ab, erst klären, nicht blind stagen.

- [ ] **Step 2: Auf einen Branch wechseln**

`main` ist der getaggte Integrationsstand — auch Dokumentation läuft über einen Branch. Uncommittete Änderungen wandern beim `switch` automatisch mit, weil der neue Branch von `main` ausgeht.

```bash
git switch -c fork-only/fork-documentation main
git status --short
```

Erwartet: dieselben fünf Einträge wie in Step 1.

- [ ] **Step 3: Stagen und Sanitize-Scan**

```bash
git add FORK.md README.md CLAUDE.md \
        reports/2026-08-03-001-fork-strategy.md \
        reports/2026-08-03-002-fork-cleanup-plan.md

git diff --cached | grep -nE \
  '\b[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\b|\b[A-Z][0-9]{2}K[0-9]{6}\b|\bDEVK[0-9]{6,}\b'
```

Erwartet: **keine Ausgabe** (grep exit 1). Bei Treffern: Inhalt nach `.local/` verschieben und durch einen synthetischen Platzhalter ersetzen.

- [ ] **Step 4: Committen**

```bash
git commit -m "docs(fork): add FORK.md, fork strategy report and implementation plan

Establishes the downstream-distribution model against oisee/vibing-steampunk:
branch layout, the two merge rules, monthly upstream sync, and the procedure
for adopting third-party upstream PRs.

FORK.md is the operational reference; the report carries the rationale.
CLAUDE.md and README.md now point at it."
```

- [ ] **Step 5: Nach `main` mergen und pushen**

```bash
git switch main
git merge --no-ff fork-only/fork-documentation
git push origin main
git branch -d fork-only/fork-documentation
```

- [ ] **Step 6: Verifizieren**

```bash
git log --oneline -3
git show --stat HEAD | head -20
```

Erwartet: ein Merge-Commit auf `main`, darunter der Docs-Commit mit fünf Dateien.

---

### Task 2: Go-Toolchain bereitstellen

Blockiert Task 3, 5 und 6. `go` ist auf dieser Maschine weder im PowerShell- noch im Bash-PATH auffindbar; die bisherigen Releases liefen offenbar über GitHub Actions.

**Files:** keine.

**Interfaces:**
- Produces: funktionierendes `go build ./...` und `go test ./...` als lokales Gate für Task 5 und 6.

- [ ] **Step 1: Installieren**

```powershell
winget install --id GoLang.Go --source winget
```

Falls winget nicht verfügbar ist: MSI von <https://go.dev/dl/> laden, Version ≥ 1.25.0 (die `go.mod` verlangt `go 1.25.0`).

- [ ] **Step 2: Neue Shell öffnen und Version prüfen**

Die PATH-Änderung greift erst in einer neuen Shell.

```powershell
go version
```

Erwartet: `go version go1.25.x windows/amd64` oder höher.

- [ ] **Step 3: Baseline-Build**

```powershell
go build ./...
```

Erwartet: keine Ausgabe, Exit 0.

- [ ] **Step 4: Baseline-Tests**

```powershell
go test ./...
```

Erwartet: alle Pakete `ok` oder `no test files`. **Schlägt hier etwas fehl, ist das eine Vorbelastung von `main`** — dann diesen Zustand festhalten, bevor Task 5 etwas merged, sonst ist später nicht unterscheidbar, wer den Test gebrochen hat.

- [ ] **Step 5: Baseline notieren**

Ergebnis von Step 4 als Ausgangslage in `FORK.md` unter *Release* vermerken, falls Tests fehlschlagen. Ist alles grün, entfällt der Eintrag.

---

### Task 3: CI-Workflow für Pull Requests

Aktuell enthält `.github/workflows/` nur `release.yml` mit `workflow_dispatch`. Es gibt also **kein automatisches Gate**. Da lokal bislang keine Toolchain vorhanden war, ist CI das verlässlichere Netz — und es macht fork-interne PRs erst sinnvoll.

**Files:**
- Create: `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: `go.mod` (Version via `go-version-file`).
- Produces: ein Statuscheck `build-test` auf jedem PR und auf `main`.

- [ ] **Step 1: Branch anlegen**

Fork-only: der Workflow ist auf den Fork-Betrieb zugeschnitten. Er kann später als eigener PR nach oben angeboten werden.

```bash
git switch -c fork-only/ci-workflow main
```

- [ ] **Step 2: Workflow schreiben**

Datei `.github/workflows/ci.yml`:

```yaml
name: CI

on:
  pull_request:
  push:
    branches: [main]

jobs:
  build-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true

      - name: Build
        run: go build ./...

      - name: Test
        run: go test ./...
```

`go vet ./...` bewusst nicht enthalten: ein rotes CI am ersten Tag wegen Vorbelastungen entwertet das Gate. Nachrüsten, sobald `go vet ./...` lokal sauber durchläuft.

- [ ] **Step 3: Lokal gegenprüfen**

```bash
go build ./... && go test ./...
```

Erwartet: identisches Ergebnis zur Baseline aus Task 2 Step 4. Der Workflow darf nichts prüfen, was lokal nicht auch grün ist.

- [ ] **Step 4: Committen**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: run build and tests on pull requests and main

The repo had no PR gate — release.yml is workflow_dispatch only. This adds
build + test on every PR and on pushes to main, so fork-internal PRs carry a
real signal.

go vet is deliberately omitted until it passes cleanly on main."
```

- [ ] **Step 5: Mergen und pushen**

```bash
git switch main
git merge --no-ff fork-only/ci-workflow
git push origin main
git branch -d fork-only/ci-workflow
```

- [ ] **Step 6: Lauf verifizieren**

```bash
gh run list --repo frd1201/vibing-steampunk --limit 3
```

Erwartet: ein `CI`-Lauf mit Status `completed` / `success`. Bei `failure`: Log ansehen mit `gh run view --log-failed` und beheben, bevor Task 5 startet — sonst fehlt dort das Gate.

---

### Task 4: Geerbte Upstream-Branches von `origin` entfernen

Elf Branches wurden beim Forken mitkopiert. Sie sind fremde Arbeit, tragen zusammen rund 36 Commits, die nicht in `main` sind, und verstellen den Blick auf die eigenen Branches. Alle elf Tips wurden am 2026-08-03 als **identisch mit `upstream/<name>`** verifiziert — es geht nichts verloren.

**Files:** keine.

**Interfaces:**
- Produces: `git branch -r --list 'origin/*'` zeigt danach nur noch `main` plus die vier eigenen Branches.

- [ ] **Step 1: Identität der Tips erneut prüfen**

Die Prüfung von 2026-08-03 wird vor dem Löschen wiederholt — zwischenzeitlich könnte gepusht worden sein.

```bash
git fetch upstream --prune
git fetch origin --prune

for b in abap-lsp chore/future-plans claude/admiring-hamilton \
         claude/fervent-curran claude/magical-galileo decompose-phase1 \
         feat/wasm-abap feature/debug-daemon-parked one-tool-mode \
         pr-93-fix worktree-integration-test-infra; do
  o=$(git rev-parse "origin/$b" 2>/dev/null)
  u=$(git rev-parse "upstream/$b" 2>/dev/null)
  if [ "$o" = "$u" ]; then echo "OK        $b"; else echo "ABWEICHEND $b  origin=$o upstream=$u"; fi
done
```

Erwartet: elf Zeilen `OK`. **Jede `ABWEICHEND`-Zeile bedeutet: diesen Branch nicht löschen**, sondern erst klären, woher der Unterschied kommt.

- [ ] **Step 2: Löschen**

Nur ausführen, wenn Step 1 elfmal `OK` lieferte.

```bash
for b in abap-lsp chore/future-plans claude/admiring-hamilton \
         claude/fervent-curran claude/magical-galileo decompose-phase1 \
         feat/wasm-abap feature/debug-daemon-parked one-tool-mode \
         pr-93-fix worktree-integration-test-infra; do
  git push origin --delete "$b"
done
```

- [ ] **Step 3: Verifizieren**

```bash
git fetch origin --prune
git branch -r --list 'origin/*'
```

Erwartet: `origin/main`, `origin/feat/incl-write-support`, `origin/fix/csrf-head-fallback-and-session-type`, `origin/fix/search-type-filter-issue-119`, `origin/fork-only/onprem-edit-fixes`, `origin/test/fork-with-pr-108`.

- [ ] **Step 4: Wiederherstellbarkeit belegen**

```bash
git log --oneline -1 upstream/one-tool-mode
```

Erwartet: der Commit ist weiterhin da. Das ist der Beleg, dass Step 2 nichts vernichtet hat.

---

### Task 5: Upstream-PR #108 übernehmen

**Entschieden am 2026-08-03: PR #108 gewinnt gegen den NoModification-Guard.**

Begründung: `1bc5804` weist nach, dass `MODIFICATION_SUPPORT="NoModification"` in SAPs `IF_ADT_LOCK_RESULT` die Konstante `CO_MOD_SUPPORT_NOT_NEEDED` trägt — „Modification-Support wird nicht *benötigt*, weil das Objekt im Kundennamensraum liegt". Nicht `NOT_PERMITTED`. Der Guard aus `22517d4` ist damit ein False Positive auf normalem Z*-Kundencode. Belegt durch Live-Traces mit anschließendem HTTP 200 auf dem PUT und durch einen Table-Test über alle vier dokumentierten SAP-Werte.

Konsequenz für `6b2cece`: dessen **Teil 2** (konfigurierbarer Guard) entfällt ersatzlos — ein Flag, das ein Nicht-Problem abschaltet. Der Best-Effort-Unlock steht **innerhalb** des Guard-Blocks (`pkg/adt/crud.go`) und fällt mit ihm. Übrig bleibt Teil 1, nachgezogen in Task 6.

Konfliktmessung vom 2026-08-03 per `git merge-tree`: `main` ← `upstream/pr-108` ergibt **zwei** Konflikte. Die umgekehrte Reihenfolge — erst `6b2cece`, dann #108 — ergäbe **vier**. Daher steht #108 vor Task 6.

**Files:**
- Modify (durch Merge): `pkg/adt/http.go` **(Konflikt)**, `pkg/adt/workflows_deploy.go` **(Konflikt)**, `pkg/adt/config.go`, `pkg/adt/crud.go`, `pkg/adt/crud_reconcile_test.go`
- Modify: `FORK.md` (Tabelle *Upstream PR decisions*)

**Interfaces:**
- Consumes: grünes `go build ./...` und `go test ./...` aus Task 2, CI-Gate aus Task 3.
- Produces:
  - `LockObject(ctx, objectURL, accessMode)` gibt das geparste `*LockResult` unverändert zurück — **kein Fehler mehr** bei `ModificationSupport == "NoModification"`. Die Signatur bleibt in dieser Task dreistellig; Task 6 erweitert sie.
  - `TestLockObject_PassesThroughModificationSupport` ersetzt `TestLockObject_RejectsNoModification`.
  - ADT-Header überleben HTTP-Redirects; opt-in Trace über `VSP_HTTP_TRACE`.
  - Veraltete `sap-contextid` wird bei `ICMENOSESSION`-Recovery verworfen.

- [ ] **Step 1: PR-Head als Branch holen**

```bash
git fetch upstream 'refs/pull/108/head:refs/remotes/upstream/pr-108'
git switch -c upstream-pr/108 upstream/pr-108
git log --oneline --no-merges -4
```

Erwartet, in dieser Reihenfolge: `ff8fd47`, `eece59c`, `1bc5804`, `8cb45a5`. Am 2026-08-03 verifiziert: das ist exakt der Inhalt von `test/fork-with-pr-108`, nur ohne dessen alten Experiment-Merge.

- [ ] **Step 2: Merge auf `main` starten**

Der Probe-Branch aus Workflow B entfällt hier: die Konfliktlage ist bereits per `git merge-tree` vermessen und die Übernahme ist beschlossen. `git merge --abort` ist der Ausstieg, falls die Auflösung schiefgeht.

```bash
git switch main
git merge --no-ff upstream-pr/108
```

Erwartet: `CONFLICT (content)` in `pkg/adt/http.go` und `pkg/adt/workflows_deploy.go`. Andere Dateien mergen automatisch.

- [ ] **Step 3: `pkg/adt/http.go` auflösen — beide Seiten behalten**

`main` bringt aus PR #120 den CSRF-`HEAD`→`GET`-Fallback samt Abbruch bei 401/403 (`29a257b`, `886a9b2`). PR #108 bringt Header-Erhalt über Redirects, den opt-in HTTP-Trace und das Verwerfen veralteter `sap-contextid`. Die beiden Änderungen sind fachlich unabhängig — **keine Seite darf gewinnen**, beide Verhalten müssen erhalten bleiben.

```bash
git diff --diff-filter=U -- pkg/adt/http.go
```

Nach der Auflösung:

```bash
git add pkg/adt/http.go
```

- [ ] **Step 4: `pkg/adt/workflows_deploy.go` auflösen — #108-Fassung als Basis**

Beide Seiten verschieben SyntaxCheck vor Lock. `8f6c030` (in `main`) tut das mit 47 geänderten Zeilen als Nebenaspekt der INCL-Arbeit. `8cb45a5` (PR #108) tut es mit 98 Zeilen und ordnet den gesamten `Lock → UpdateSource → Unlock → Activate`-Block als ununterbrochene stateful Sequenz — das ist die gründlichere Fassung und wird als Basis genommen.

**Kritisch: die INCL-Behandlung aus `8f6c030` muss dabei erhalten bleiben.** Gegenprobe:

```bash
git show 8f6c030 -- pkg/adt/workflows_deploy.go
git diff --diff-filter=U -- pkg/adt/workflows_deploy.go
git add pkg/adt/workflows_deploy.go
```

- [ ] **Step 5: Merge abschließen**

```bash
git commit -m "Merge upstream PR #108 (dme007) — deploy session ordering + MODIFICATION_SUPPORT pass-through

Adopts the PR in full. 1bc5804 removes the NoModification guard added by
22517d4: SAP's IF_ADT_LOCK_RESULT documents the value as
CO_MOD_SUPPORT_NOT_NEEDED, so the guard fired as a false positive on
customer-namespace objects.

Conflicts resolved by keeping both sides in http.go (our CSRF HEAD->GET
fallback from #120 plus their redirect header preservation and contextid
recovery), and by taking their fuller workflows_deploy.go rewrite while
retaining our INCL handling from 8f6c030."
```

- [ ] **Step 6: Gate**

```bash
go build ./... && go test ./...
go test -run 'TestLockObject' -v ./pkg/adt/
```

Erwartet: grün, darunter `TestLockObject_PassesThroughModificationSupport`. `TestLockObject_RejectsNoModification` existiert nicht mehr — das ist der beabsichtigte Effekt, kein Fehler.

- [ ] **Step 7: INCL-Pfad gezielt nachtesten**

Die Auflösung in Step 4 kann die INCL-Arbeit beschädigt haben, ohne dass ein Test darauf anschlägt.

```bash
go test -run 'INCL|Include' -v ./pkg/adt/
```

Erwartet: grün. **Kein Treffer bedeutet fehlende Abdeckung, nicht Erfolg** — dann Step 4 manuell gegen `git show 8f6c030 -- pkg/adt/workflows_deploy.go` gegenlesen.

- [ ] **Step 8: `FORK.md` nachziehen**

Die Zeile für #108 in *Upstream PR decisions* auf `**adopted**` setzen, Begründung eintragen. Zusätzlich #125 auf `**superseded**` setzen: dessen Thema (redundantes Mutation-Gate nach Lock) ist durch #108 mitbehandelt.

- [ ] **Step 9: Committen und pushen**

```bash
git add FORK.md
git commit -m "docs(fork): record adoption of upstream PR #108"
git push origin main
```

- [ ] **Step 10: Alten Experiment-Branch entfernen**

`test/fork-with-pr-108` hat keinen offenen PR und ist durch `upstream-pr/108` ersetzt.

```bash
git push origin --delete test/fork-with-pr-108
```

- [ ] **Step 11: CI verifizieren**

```bash
gh run list --repo frd1201/vibing-steampunk --limit 2
```

Erwartet: `success`.

---

### Task 6: `corrNr` beim LOCK nachziehen (Teil 1 von `6b2cece`)

Was von `6b2cece` nach der Entscheidung in Task 5 übrig bleibt: die 4-arg-Signatur von `LockObject`, das Aussenden von `corrNr` bei gesetztem Transport, die 22 angepassten Aufrufstellen und das Durchreichen des echten Transports in den Write-Workflows. Rückwärtskompatibel — `transport == ""` verhält sich wire-level identisch zu vorher. **In PR #108 steht davon nichts.**

Ersatzlos verworfen wird alles, was nur den Guard bedient hat: `SafetyConfig.IgnoreNoModificationGuard`, das CLI-Flag `--ignore-no-modification-guard`, die Env-Variable `SAP_IGNORE_NO_MODIFICATION_GUARD`, die Verdrahtung in `internal/mcp/server.go` und `cmd/vsp/main.go`, der README-Abschnitt dazu, sowie die beiden Tests `TestLockObject_BypassesGuardWhenConfigured` und `TestLockObject_UnlocksOnGuardError`.

Der überlebende Teil hat in `6b2cece` **keinen Test** — die Tests dort deckten ausschließlich den Guard ab. Diese Task legt ihn deshalb per TDD an, und muss dafür zuerst den Mock erweitern: `recordedCall` zeichnet bislang nur Methode und Pfad auf, nicht den Query-String.

**Files:**
- Modify: `pkg/adt/crud.go` (Signatur + `corrNr`), `pkg/adt/workflows.go`, `pkg/adt/workflows_source.go`, `pkg/adt/workflows_edit.go`, `pkg/adt/workflows_deploy.go`, `pkg/adt/workflows_execute.go`, `pkg/adt/workflows_fileio.go`, `internal/mcp/handlers_crud.go`, `internal/mcp/handlers_deploy.go`
- Test: `pkg/adt/crud_reconcile_test.go`

**Interfaces:**
- Consumes: `main` nach Task 5 (Guard entfernt).
- Produces: `func (c *Client) LockObject(ctx context.Context, objectURL string, accessMode string, transport string) (*LockResult, error)` — **Signaturänderung von drei auf vier Parameter.** Bei `transport != ""` wird der Query-Parameter `corrNr=<transport>` gesetzt, sonst gar nicht.

- [ ] **Step 1: Branch anlegen**

```bash
git switch -c fork-only/corrnr-at-lock main
```

- [ ] **Step 2: Mock um den Query-String erweitern**

In `pkg/adt/crud_reconcile_test.go`, `recordedCall` (Zeile ~33) und `Do` (Zeile ~39):

```go
type recordedCall struct {
	method string
	path   string
	query  string
}
```

```go
func (m *methodPathMock) Do(req *http.Request) (*http.Response, error) {
	m.calls = append(m.calls, recordedCall{
		method: req.Method,
		path:   req.URL.Path,
		query:  req.URL.RawQuery,
	})
```

- [ ] **Step 3: Den fehlschlagenden Test schreiben**

Ans Ende von `pkg/adt/crud_reconcile_test.go`:

```go
// TestLockObject_EmitsCorrNr pins the corrNr query parameter that on-premise
// SAP systems expect when locking an object in a transportable package.
// transport == "" must stay wire-level identical to the pre-4-arg behaviour.
func TestLockObject_EmitsCorrNr(t *testing.T) {
	tests := []struct {
		name      string
		transport string
		wantCorr  bool
	}{
		{name: "transportable package", transport: "TR-EXAMPLE", wantCorr: true},
		{name: "no transport", transport: "", wantCorr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &methodPathMock{
				routes: []routedResponse{
					resp("", "discovery", 200, "ok"),
					resp(http.MethodPost, "/oo/classes/ZCL_DEMO", 200, lockResponseXML),
				},
			}
			cfg := NewConfig("https://sap.example.com:44300", "user", "pass")
			client := NewClientWithTransport(cfg, NewTransportWithClient(cfg, mock))

			_, err := client.LockObject(
				context.Background(),
				"/sap/bc/adt/oo/classes/ZCL_DEMO",
				"MODIFY",
				tt.transport,
			)
			if err != nil {
				t.Fatalf("LockObject returned %v, want nil", err)
			}

			var lockQuery string
			var seen bool
			for _, c := range mock.calls {
				if c.method == http.MethodPost && strings.Contains(c.path, "ZCL_DEMO") {
					lockQuery, seen = c.query, true
				}
			}
			if !seen {
				t.Fatal("no LOCK request was recorded")
			}

			gotCorr := strings.Contains(lockQuery, "corrNr=TR-EXAMPLE")
			if gotCorr != tt.wantCorr {
				t.Errorf("query = %q, corrNr present = %v, want %v", lockQuery, gotCorr, tt.wantCorr)
			}
			if !tt.wantCorr && strings.Contains(lockQuery, "corrNr") {
				t.Errorf("query = %q, must not emit corrNr when transport is empty", lockQuery)
			}
		})
	}
}
```

- [ ] **Step 4: Test laufen lassen — muss fehlschlagen**

```bash
go test -run TestLockObject_EmitsCorrNr ./pkg/adt/
```

Erwartet: Compile-Fehler `too many arguments in call to client.LockObject` — die Signatur hat noch drei Parameter. Das ist der korrekte Fehlschlag.

- [ ] **Step 5: `6b2cece` ohne Commit anwenden**

```bash
git cherry-pick -n 6b2cece
```

Konflikte sind zu erwarten, weil `main` nach Task 5 den Guard-Block nicht mehr enthält, auf den sich `6b2cece` bezieht.

- [ ] **Step 6: Guard-Anteile verwerfen**

Diese Dateien werden aus dem Cherry-Pick vollständig zurückgesetzt — sie enthalten ausschließlich Guard-Verdrahtung bzw. vier Monate alte Doku-Stände, die mit den Änderungen aus Task 1 kollidieren:

```bash
git checkout HEAD -- pkg/adt/safety.go internal/mcp/server.go cmd/vsp/main.go README.md CLAUDE.md
```

In `pkg/adt/crud.go` bleibt **nur** erhalten:
- die Signaturänderung auf vier Parameter samt Doc-Kommentar,
- der Block `if transport != "" { params.Set("corrNr", transport) }`,
- die auf `""` umgestellten Aufrufstellen in `tryCleanupOrphanLock`, `cleanupPartialObject`, `CreateTable`.

In `pkg/adt/crud_reconcile_test.go` werden `TestLockObject_BypassesGuardWhenConfigured` und `TestLockObject_UnlocksOnGuardError` **nicht** übernommen.

- [ ] **Step 7: Prüfen, dass kein Guard-Rest übrig ist**

```bash
grep -rn "IgnoreNoModificationGuard\|ignore-no-modification-guard\|SAP_IGNORE_NO_MODIFICATION_GUARD" \
  --include='*.go' --include='*.md' .
```

Erwartet: **keine Ausgabe.** Jeder Treffer ist ein Rest der verworfenen Guard-Verdrahtung.

- [ ] **Step 8: Test laufen lassen — muss bestehen**

```bash
go test -run TestLockObject_EmitsCorrNr -v ./pkg/adt/
```

Erwartet: beide Subtests `PASS`.

- [ ] **Step 9: Volles Gate**

```bash
go build ./... && go test ./...
```

Erwartet: grün. Die Signaturänderung berührt 22 Aufrufstellen — ein Compile-Fehler hier bedeutet eine übersehene Stelle.

- [ ] **Step 10: Committen — bereinigte Message**

Der ursprüngliche Commit trug einen Agenten-Co-Author-Trailer und im Body einen Systemalias, der ein Live-System benennt. Beides darf nach den Global Constraints nicht in die Historie — die Message unten ist bereits bereinigt.

```bash
git add -A
git commit -m "feat(adt): pass corrNr at LOCK time

LockObject takes a fourth parameter, the transport task number, and emits
it as the corrNr query parameter. transport=\"\" produces no corrNr and is
wire-level identical to the previous behaviour, so all 22 internal call
sites keep working; the write workflows pass the real transport.

This matches the SAP ADT API spec and covers on-premise systems that
expect corrNr already at lock time rather than only on the PUT.

Verified end-to-end against an on-premise system: EditSource on a class in
a transportable package succeeds.

Part 1 of the parked fork-only/onprem-edit-fixes branch. Part 2 of that
branch — a configurable MODIFICATION_SUPPORT guard — is deliberately
dropped: upstream PR #108 removed the guard entirely, so the flag would
only have disabled a false positive."
```

- [ ] **Step 11: Nach `main` mergen**

```bash
git switch main
git merge --no-ff fork-only/corrnr-at-lock
go build ./... && go test ./...
git push origin main
git branch -d fork-only/corrnr-at-lock
```

- [ ] **Step 12: Geparkten Branch auflösen**

Sein verwertbarer Inhalt steckt jetzt in `main`, kein PR hängt daran.

```bash
git push origin --delete fork-only/onprem-edit-fixes
```

- [ ] **Step 13: `FORK.md` nachziehen**

In *Fork-only changes* ergänzen:

```markdown
| `fork-only/corrnr-at-lock` | corrNr at LOCK time (4-arg LockObject) | on-prem transportable-package edits; not covered by upstream PR #108 |
```

Dazu in *Known SHA-tracking gaps* eine Zeile: `6b2cece` wurde geteilt übernommen, Teil 2 verworfen — der ursprüngliche Commit existiert in `main` nicht mehr als SHA.

```bash
git add FORK.md
git commit -m "docs(fork): record corrNr adoption and the dropped guard"
git push origin main
```

---

### Task 7: Upstream-PRs nachfassen und Termine setzen

Abschluss: die drei eigenen PRs anstoßen und die beiden datumsgebundenen Entscheidungen aus der Strategie so ablegen, dass sie nicht vergessen werden.

**Files:**
- Modify: `FORK.md` (Tabelle *Our open upstream PRs*)

**Interfaces:**
- Consumes: `FORK.md` aus Task 1.

- [ ] **Step 1: Bei den drei eigenen PRs nachfassen**

Ein Kommentar pro PR, sachlich, kein Drängen.

```bash
for n in 120 121 126; do
  gh pr comment "$n" --repo oisee/vibing-steampunk \
    --body "Friendly ping — this is still relevant and rebases cleanly on main. Happy to adjust anything if you'd like changes. No rush."
done
```

- [ ] **Step 2: Reaktion prüfen**

```bash
gh pr list --repo oisee/vibing-steampunk --author frd1201 --state all \
  --json number,state,updatedAt,title
```

Erwartet: alle drei weiterhin `OPEN`. Die Schließfrist (rund zwölf Monate ohne Antwort) fällt damit auf **2027-04-23** für #120 und #121 und auf **2027-05-01** für #126.

- [ ] **Step 3: Beide Termine in `FORK.md` verankern**

Unter *Our open upstream PRs* ergänzen:

```markdown
**Review dates:** module-path trigger check on **2026-10-15** (see strategy report,
section 7). Close-if-unanswered dates: #120 and #121 on **2027-04-23**, #126 on
**2027-05-01**.
```

- [ ] **Step 4: Kalendereintrag setzen**

Die Prüfung am **2026-10-15** entscheidet über den Modulpfad-Umzug (104 Dateien). Ein Eintrag in `FORK.md` allein erinnert niemanden — zusätzlich einen echten Termin anlegen.

- [ ] **Step 5: Committen und pushen**

```bash
git add FORK.md
git commit -m "docs(fork): add review dates for module path trigger and PR closure"
git push origin main
```

- [ ] **Step 6: Endzustand verifizieren**

```bash
git fetch origin --prune && git fetch upstream --prune
git branch -r --list 'origin/*'
git rev-list --left-right --count upstream/main...main
git status --short
```

Erwartet: `origin` trägt nur noch `main` plus die drei Branches mit offenem PR; `main` ist weiterhin `0` hinter Upstream; der Working Tree ist sauber.

---

## Was dieser Plan bewusst nicht tut

- **Kein Modulpfad-Umzug.** Entscheidung vertagt auf 2026-10-15 (Strategie, Abschnitt 7).
- **Keine Entscheidung zu #139 und #145.** Beide Kollisionen brauchen einen konkreten Anlass nach Workflow B. Sie stehen als `pending` in `FORK.md` und werden einzeln behandelt, wenn ein Problem sie auslöst. #125 wird in Task 5 als `superseded` geschlossen, weil #108 dasselbe Thema mitbehandelt.
- **Keine erneute Prüfung der Guard-Entscheidung.** Dass `MODIFICATION_SUPPORT="NoModification"` als `CO_MOD_SUPPORT_NOT_NEEDED` durchgereicht statt abgewiesen wird, ist am 2026-08-03 auf Basis von dme007's Nachweis entschieden worden. Sollte auf einem BTP- oder Cloud-System doch ein unklarer 423 auftreten, ist das der Anlass, die Entscheidung neu aufzurollen — nicht dieser Plan.
- **Keine `release/*`-Branches.** Erst sinnvoll, wenn mehrere Personen parallel gegen verschiedene Systeme testen.
- **Keine Reparatur von `a47b225` und `886a9b2`.** Inhalt korrekt, nur die SHA-Verfolgung reißt; dokumentiert statt umgeschrieben.
- **Kein `go vet` im CI.** Erst nachrüsten, wenn es lokal sauber durchläuft.
