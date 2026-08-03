# Fork-Strategie: Weiterentwicklung und Upstream-Übernahme

**Datum:** 2026-08-03
**Status:** verabschiedet
**Gilt für:** `frd1201/vibing-steampunk` (Fork) gegenüber `oisee/vibing-steampunk` (Upstream)

---

## 1. Ausgangslage

| Fakt | Stand am 2026-08-03 |
|---|---|
| Fork | `frd1201/vibing-steampunk` |
| Upstream | `oisee/vibing-steampunk` |
| `main` gegenüber `upstream/main` | 15 Commits voraus, 0 zurück |
| Letzter Upstream-Code-Commit | 2026-04-15 |
| Letzter Upstream-Commit überhaupt | 2026-06-15 (nur Docs) |
| Letztes Upstream-Tag | `v2.38.1` |
| Fork-eigene Tags | `v3.0.0`, `v3.0.1` |
| Eigene offene Upstream-PRs | #120, #121, #126 (seit April/Mai unbeantwortet) |
| Fremde offene Upstream-PRs | 12 |
| Modulpfad in `go.mod` | `github.com/oisee/vibing-steampunk` |
| Release-Ziel in `.goreleaser.yml` | `frd1201` |

### Bewertung des bisherigen Vorgehens

Das bisherige Vorgehen war in den entscheidenden Punkten korrekt:

- **Es wurde gemergt, nicht rebased** (`3da393b Merge branch 'oisee:main' into main`).
  Für einen Fork, der Upstream weiter verfolgen will, ist das die richtige Wahl.
  Ein Rebase hätte die Historie umgeschrieben und jeden künftigen Merge verteuert.
- **Features entstanden auf Feature-Branches und gingen zuerst als PR nach oben**,
  dann in den eigenen `main`. Dieses Muster hält die Rückführbarkeit offen.
- **`main` ist nicht hinter Upstream.** Es besteht keine aufgelaufene Merge-Schuld.

Drei Schwachstellen wurden gefunden:

1. **Inkonsistente Identität.** `.goreleaser.yml` released nach `frd1201` und es
   werden eigene Versionen getaggt, aber `go.mod` trägt weiterhin den
   Upstream-Modulpfad. Solange die Verteilung über Klonen/Bauen aus dem eigenen
   Repo läuft, ist das unschädlich — siehe Abschnitt 7.
2. **Cherry-Pick statt Merge.** `a47b225` und `886a9b2` in `main` sind
   inhaltsgleiche Kopien von `2ea6004` und `59b401b` auf den PR-Branches:
   gleiche Message, andere SHA. Git kann diese Arbeit dadurch nicht mehr als
   erledigt ausweisen — die Frage „was liegt in `main` ohne Upstream-PR?" ist
   nur noch über einen patch-id-Abgleich beantwortbar statt über ein einfaches
   `git branch --merged` (siehe Abschnitt 9.1).
3. **Verwaiste Fork-Arbeit.** `fork-only/onprem-edit-fixes` und
   `test/fork-with-pr-108` liegen seit April unentschieden herum.

---

## 2. Grundsatzentscheidungen

| Entscheidung | Wahl | Begründung |
|---|---|---|
| Verhältnis zu Upstream | **Downstream-Distribution** | Eigenes Tempo und eigene Releases, aber dauerhaft merge-fähig. Neue Features gehen weiter als PR nach oben; fremde Upstream-PRs werden kuratiert übernommen. Die Rückkehr bleibt möglich, falls Upstream wieder aktiv wird. |
| Verteilung | **Klonen/Bauen aus dem eigenen Repo** plus eigene Release-Artefakte | Der interne Zugriff auf das Fork-Repo ist gegeben. `go install` über die URL wird nicht gebraucht. |
| Modulpfad | **bleibt `github.com/oisee/vibing-steampunk`** | Ein Umzug kostet 104 Dateien (davon 73 Go-Dateien) und danach dauerhaft Konflikte bei jedem Upstream-Merge. Das widerspricht dem Ziel „merge-fähig bleiben". Trigger für eine spätere Neubewertung siehe Abschnitt 7. |
| Umgang mit fremden Upstream-PRs | **bedarfsgetrieben und kuratiert** | Nur übernehmen, was für die eigenen SAP-Systeme tatsächlich gebraucht wird. Geringster Aufwand, geringste Konfliktfläche. |
| Branch-Modell | **Integrations-`main` mit Merge-Tracking** | Bestätigt die bestehende Arbeitsweise. Rebase scheidet aus, weil `main` getaggte Releases trägt und intern daraus gebaut wird — force-push ist dort nicht vertretbar. |

---

## 3. Branch-Layout

| Präfix | Zweck | Zweigt ab von | Landet in |
|---|---|---|---|
| `main` | Integrationsstand, Quelle aller Tags | — | — |
| `feat/*`, `fix/*` | eigene Entwicklung, **upstream-tauglich** | `upstream/main` | PR nach upstream **und** Merge nach `main` |
| `fork-only/*` | bewusst nicht upstream-tauglich | `main` | nur Merge nach `main` |
| `upstream-pr/<nr>` | Übernahme eines fremden Upstream-PRs | Head des jeweiligen PRs | Merge nach `main` |
| `sync/upstream-YYYY-MM` | Auffangbranch für einen Upstream-Merge | `main` | Merge nach `main` |
| `probe/pr-<nr>` | Wegwerf-Branch für einen Test-Merge | `main` | wird gelöscht |
| `release/*` | vorgemerkt, heute nicht aktiv | `main` | — |

### Die zwei tragenden Regeln

**Regel 1 — Upstream-Taugliches zweigt von `upstream/main` ab, nicht von `main`.**
Andernfalls schleppt der PR den Fork-Stand mit und wird für den Upstream-Maintainer
unnötig groß. Der `Merge branch 'oisee:main'` in
`fix/csrf-head-fallback-and-session-type` ist ein Beispiel dafür.

**Regel 2 — nie cherry-picken, immer mergen.**
Ein Cherry-Pick erzeugt einen inhaltsgleichen Commit mit anderer SHA. Der
Schaden ist dabei nicht in erster Linie ein Merge-Konflikt: bei identischem
Inhalt löst Gits Drei-Wege-Merge das in aller Regel stillschweigend auf. Der
Schaden ist der **Verlust der Nachvollziehbarkeit** — `git log upstream/main..main`
und `git branch --merged` sagen nicht mehr die Wahrheit darüber, was bereits
nach oben eingereicht ist, und die Frage „was in unserem `main` steckt in keinem
Upstream-PR?" lässt sich nur noch über einen patch-id-Abgleich beantworten
(Abschnitt 9.1).

Ein Merge dagegen bringt die Original-Commits mit ihren echten SHAs in die
Historie; merged Upstream später dieselben Commits, erkennt Git sie als
gemeinsame Vorfahren. Zum echten Konflikt wird ein Cherry-Pick erst, wenn die
beiden Fassungen auseinanderlaufen — etwa weil der Upstream-Maintainer beim
Merge noch etwas angepasst hat. Das ist der seltenere, aber teurere Fall.

### `release/*` — bewusst vertagt

Ein zweistufiges Modell (`main` als Integration, `release/3.1` als getesteter
Stand, von dem getaggt wird) wird sinnvoll, sobald mehrere Personen parallel
gegen verschiedene SAP-Systeme testen. Im aktuellen Betrieb ist es Overhead
ohne Gegenwert. Als Option vorgemerkt, heute nicht eingeführt.

---

## 4. Upstream-Sync

Ein Fork bricht nicht daran, dass Upstream schläft, sondern daran, dass niemand
nachschaut und die Merge-Schuld unbemerkt anwächst. „Stale" ist nicht „tot" —
Upstream hat im Juni noch committet.

### Rhythmus: monatlich, oder wenn GitHub Aktivität meldet

```bash
git fetch upstream --prune
git log --oneline main..upstream/main        # leer? Fertig, nichts zu tun.
```

Ist die Liste leer, kostet der Check zwei Minuten. Ist sie nicht leer, läuft der
Merge über einen Auffangbranch statt direkt auf `main`:

```bash
git switch -c sync/upstream-2026-08 main
git merge upstream/main
go build ./... && go test ./...              # Gate: muss grün sein
git switch main && git merge --no-ff sync/upstream-2026-08
git branch -d sync/upstream-2026-08
```

Der Auffangbranch kostet einen Befehl und erspart es, einen fehlgeschlagenen
Merge auf `main` rückabwickeln zu müssen — auf einem Branch, von dem getaggt
wird und aus dem intern gebaut wird, ist das den Aufwand wert.

---

## 5. Ablauf A: eigene Weiterentwicklung

Eine einzige Frage entscheidet über den Branch-Typ:

> Löst die Änderung ein Problem, das jeder Nutzer hat, und enthält sie nichts
> Monads- oder kundenspezifisches?

### Ja — upstream-tauglich

```bash
git fetch upstream
git switch -c feat/<thema> upstream/main       # nicht von main!
# ... entwickeln, testen ...
git push -u origin feat/<thema>
gh pr create --repo oisee/vibing-steampunk --base main
git switch main && git merge --no-ff feat/<thema>
```

**Es wird nicht auf den Upstream-Merge gewartet.** Die Änderung geht sofort in
den eigenen `main`, damit sie produktiv verfügbar ist. Bei einem stale Upstream
wäre Warten gleichbedeutend mit Nichtstun.

**Der PR-Branch wird nicht gelöscht, solange der PR offen ist** — ein gelöschter
Head-Branch schließt den PR auf GitHub automatisch.

### Nein — fork-only

```bash
git switch -c fork-only/<thema> main
# ... entwickeln, testen ...
git switch main && git merge --no-ff fork-only/<thema>
```

Typische Fälle: On-Prem-Spezifika, Release-Targets, interne Konfiguration,
alles, was auf ein konkretes Systemumfeld zugeschnitten ist.

### Nachträgliches Upstreamen

Landet doch einmal etwas Upstream-Taugliches in `main`, ohne dass ein PR dafür
existiert, wird es so nachgereicht:

```bash
git switch -c feat/<thema> upstream/main
git cherry-pick <sha>                          # hier unvermeidbar
go build ./... && go test ./...
git push -u origin feat/<thema>
gh pr create --repo oisee/vibing-steampunk --base main
```

Das ist die **einzige Ausnahme von Regel 2** — und zugleich der Grund, warum
Regel 1 existiert: wer von Anfang an von `upstream/main` abzweigt, braucht
diesen Nachbau nie. Der Preis des Nachreichens ist genau der
Traceability-Verlust aus Abschnitt 9.1, denn die Änderung steht danach mit zwei
verschiedenen SHAs in der Historie.

---

## 6. Ablauf B: fremden Upstream-PR übernehmen

**Auslöser ist immer ein konkretes Problem**, für das oben bereits eine Lösung
liegt — nicht „der PR sieht nützlich aus".

```bash
gh pr checkout <nr> --repo oisee/vibing-steampunk --branch upstream-pr/<nr>

# 1. Diff lesen, nicht überfliegen
git diff upstream/main...upstream-pr/<nr>

# 2. Test-Merge auf einem Wegwerf-Branch, Konflikte bewerten
git switch -c probe/pr-<nr> main
git merge upstream-pr/<nr>

# 3. Gate
go build ./... && go test ./...
go test -tags=integration -v ./pkg/adt/        # gegen ein echtes System

# 4. Übernahme mit Herkunft in der Merge-Message
git switch main
git merge --no-ff upstream-pr/<nr> \
  -m "Merge upstream PR #<nr> (<autor>) — <kurzbeschreibung>"
git branch -D probe/pr-<nr>
```

Jede Übernahme bekommt einen CHANGELOG-Eintrag unter „Aus Upstream-PRs
übernommen", mit PR-Nummer und Autor. Zusätzlich wird die Entscheidung in der
Tabelle in `FORK.md` festgehalten.

### Kollisionsfall

Fasst ein fremder PR denselben Code an wie eigene Arbeit, wird **nicht gemergt,
sondern zuerst entschieden**. Drei mögliche Ausgänge:

| Ausgang | Vorgehen |
|---|---|
| Eigene Lösung gewinnt | PR nicht übernehmen. Als „bewusst abgelehnt" mit Begründung in `FORK.md` eintragen. Ein sachlicher Kommentar im Upstream-PR hilft dem Maintainer bei der Entscheidung. |
| Fremde Lösung gewinnt | Eigene Änderung zurückbauen, PR übernehmen, eigenen Upstream-PR mit Verweis schließen. |
| Beide haben teilweise recht | Neuer `feat/*`-Branch von `upstream/main`, der beides vereint, als neuer PR nach oben. Teuerste Variante, aber die einzige, die den Konflikt auflöst statt ihn zu verschieben. |

Die Entscheidung wird immer mit Begründung dokumentiert. Ohne das trifft sie in
sechs Monaten jemand erneut und anders.

### Bekannte Kollisionen (Stand 2026-08-03)

| Upstream-PR | Thema | Kollidiert mit |
|---|---|---|
| #139 | Program-Includes in EditSource | eigenem #121 (INCL write support) |
| #125 | Redundantes Mutation-Gate nach Lock überspringen | `22517d4` (Lock-Handle-Fix), #120 |
| #108 | Deploy-Session-Ordering, MODIFICATION_SUPPORT | `22517d4`, `8f6c030` |
| #145 | Offenen Transport bei Write wiederverwenden statt 409 | angrenzend an die Write-Pfade |

Eigenständig und ohne bekannte Kollision: #150 (ActivateMultiple), #149 (SRVB
read), #148 (Activation-Parsing), #138 (InstallZADTVSP), #130 (ENHO read),
#128 (Browser-Auth Client), #107 (WebSocket-Proxy), #106 (Install-Description).

---

## 7. Identität, Release und Versionierung

### Modulpfad

`go.mod` bleibt auf `github.com/oisee/vibing-steampunk`. Der Umzug auf einen
eigenen Pfad kostet einmalig 104 Dateien und danach dauerhaft Konflikte bei
jedem Upstream-Merge — das steht dem Ziel „merge-fähig bleiben" entgegen. Da
die Verteilung über Klonen/Bauen aus dem eigenen Repo läuft, wird `go install`
über die URL nicht gebraucht.

**Trigger für eine Neubewertung** — tritt einer der folgenden Fälle ein, wird
der Modulpfad in einem einzigen Commit umgezogen:

- Upstream bleibt **sechs Monate ohne Code-Commit** (Frist läuft ab 2026-04-15,
  also fällig zur Prüfung am **2026-10-15**), **oder**
- die eigenen Upstream-PRs werden abgelehnt, **oder**
- die Entscheidung fällt bewusst zugunsten eines Hard Forks.

### Versionierung

Das Fork-Band ist **3.x**. Der Sprung von Upstreams 2.x auf 3.0.0 war richtig:
er macht die eigene Linie unmissverständlich und kollidiert nicht mit
Upstream-Tags.

- **Regel:** Solange Upstream im 2.x-Band bleibt, gehört 3.x dem Fork.
- **Trigger:** Taggt Upstream selbst ein 3.x, wechselt der Fork auf ein
  eindeutiges Schema in der Form `v3.4.0-fork.1`.

### Release-Gate

Getaggt wird ausschließlich von `main`, und nur wenn:

1. `go build ./...` und `go test ./...` grün sind, **und**
2. ein Integrationslauf gegen ein echtes SAP-System durchgelaufen ist.

### CHANGELOG

Pro Release zwei getrennte Abschnitte:

```markdown
## v3.1.0

### Eigene Änderungen
- ...

### Aus Upstream übernommen
- PR #145 (zooloo303): offenen Transport bei Write wiederverwenden
```

Ohne diese Trennung ist nach einem halben Jahr nicht mehr nachvollziehbar, was
von wem stammt.

---

## 8. Eigene offene Upstream-PRs

**#120, #121 und #126 bleiben offen.** Sie kosten nichts und halten die
Rückführung offen — das ist der eigentliche Wert der Downstream-Strategie.

- Einmal freundlich nachfassen: ein Kommentar pro PR, kein Drängen. Danach in
  Ruhe lassen.
- Die zugehörigen Branches **nicht löschen**, solange die PRs offen sind.
  Betroffen: `feat/incl-write-support`, `fix/csrf-head-fallback-and-session-type`,
  `fix/search-type-filter-issue-119`.
- Bleibt ein PR rund zwölf Monate unbeantwortet: mit einem sachlichen Verweis
  auf den entsprechenden Fork-Commit schließen. Das hält den Upstream-Tracker
  sauber und ist ein Signal, kein Affront.

---

## 9. Offene Aufräumarbeiten

| # | Sache | Zu tun | Status |
|---|---|---|---|
| 1 | `fork-only/onprem-edit-fixes` → `6b2cece` „corrNr at LOCK + configurable NoModification guard" | Entscheiden: nach `main` mergen oder verwerfen. Liegt seit 2026-04-27 unentschieden. Der Inhalt klingt nach produktivem Bedarf. | offen |
| 2 | `test/fork-with-pr-108` (dme007, 4 Commits + 1 Merge) | Entscheiden: als `upstream-pr/108` sauber übernehmen oder löschen. Aktuell Rauschen — und `main` hat mit `8f6c030` einen Teil davon (SyntaxCheck vor Lock) eigenständig nachgebaut, was die Übernahme verteuert. | offen |
| 3 | Gecherry-pickte `a47b225`, `886a9b2` | Nicht reparieren — der Inhalt ist korrekt und identisch. In `FORK.md` als Stellen notieren, an denen die SHA-Verfolgung reißt, damit ein Sync nach einem Upstream-Merge von #120/#121 nicht überrascht. | offen |
| 4 | `go.mod` zeigt auf `oisee` | Trigger-Prüfung am 2026-10-15 terminieren (siehe Abschnitt 7). | offen |
| 5 | 11 beim Forken mitkopierte Upstream-Branches auf `origin` | Löschen. Alle elf Tips sind mit `upstream/<name>` **identisch** (geprüft 2026-08-03), es geht also nichts verloren — nach einem `git fetch upstream` sind sie jederzeit wieder da. Betrifft `abap-lsp`, `chore/future-plans`, `claude/admiring-hamilton`, `claude/fervent-curran`, `claude/magical-galileo`, `decompose-phase1`, `feat/wasm-abap`, `feature/debug-daemon-parked`, `one-tool-mode`, `pr-93-fix`, `worktree-integration-test-infra`. | offen |

### 9.1 Deckungsanalyse — was liegt in `main` ohne Upstream-PR?

Ermittelt am 2026-08-03 über patch-id-Abgleich zwischen `upstream/main..main`
und den drei PR-Branches:

| Commit in `main` | Betreff | Deckung |
|---|---|---|
| `29a257b` | CSRF HEAD→GET fallback + SAP_SESSION_TYPE | PR #120 |
| `886a9b2` | skip CSRF GET fallback on 401/403 | PR #120 |
| `bf3b569` | INCL (PROG/I) write support | PR #121 |
| `8f6c030` | INCL-Name aus Dateiname, SyntaxCheck vor Lock | PR #121 |
| `a47b225` | Tests für `normalizeObjectURLForPackageCheck` | PR #121 |
| `5ddb308` | `--type` server-seitig an ADT | PR #126 |
| `f1f71d5` | server-seitiger Type-Filter | PR #126 |
| `8112729` | TODO für INCL canonical type | PR #126 |
| `569e39f` | MCP-Pfad verdrahten, `CanonicalObjectType` nach `adt` | PR #126 |
| `d752536` | CHANGELOG für v3.0.0 | fork-only von Natur aus |
| `3f7a90c` | goreleaser-Release-Ziel auf den Fork | fork-only, darf nicht nach oben |

**Ergebnis: keine ungedeckte Substanz.** Alle neun inhaltlichen Commits stecken
in einem der drei offenen Upstream-PRs. Die beiden übrigen sind ihrer Natur nach
fork-only — die eigene CHANGELOG-Linie und das Release-Ziel gehören nicht nach
oben. Es besteht kein Nachhol-PR-Bedarf.

Die Analyse ist wiederholbar (Git Bash). Die Branch-Liste muss aktuell gehalten
werden, wenn weitere Upstream-PRs eröffnet werden:

```bash
git fetch upstream --prune

for b in feat/incl-write-support \
         fix/csrf-head-fallback-and-session-type \
         fix/search-type-filter-issue-119; do
  git rev-list --no-merges "upstream/main..origin/$b"
done | while read -r s; do git show "$s" | git patch-id --stable; done \
     | cut -d' ' -f1 | sort -u > /tmp/pr-patches

git rev-list --no-merges "upstream/main..main" | while read -r s; do
  id=$(git show "$s" | git patch-id --stable | cut -d' ' -f1)
  grep -q "$id" /tmp/pr-patches || git log -1 --oneline "$s"
done
```

Sobald Regel 2 durchgängig eingehalten wird, ersetzt `git branch --merged` diese
Prozedur — der patch-id-Abgleich ist nur nötig, solange gecherry-pickte Commits
in der Historie stehen.

---

## 10. Dokumentation

| Datei | Inhalt |
|---|---|
| `reports/2026-08-03-001-fork-strategy.md` | Dieses Dokument: Entscheidungen, Begründungen, Historie. |
| `FORK.md` (Repo-Root) | Operative Kurzreferenz: Remotes, Branch-Präfixe, Sync-Befehle zum Kopieren, laufend gepflegte Tabelle der übernommenen und abgelehnten Upstream-PRs mit Begründung. Die Datei, die im Alltag geöffnet wird. |
| `CLAUDE.md` | Kurzer Verweis auf `FORK.md`, damit künftige Sessions das Modell kennen. |
| `README.md` | Ein Satz: Fork von `oisee/vibing-steampunk` mit eigener 3.x-Release-Linie. Fairness gegenüber Upstream und Klarheit für Nutzer. |

`FORK.md`, `CLAUDE.md` und `README.md` sind noch nicht angelegt bzw. angepasst —
sie folgen nach Freigabe dieses Dokuments.

---

## 11. Zusammenfassung in fünf Sätzen

1. Der Fork bleibt eine Downstream-Distribution: eigenes Tempo, eigene Releases,
   aber dauerhaft merge-fähig zu Upstream.
2. Upstream-taugliche Arbeit zweigt von `upstream/main` ab, geht als PR nach
   oben und sofort in den eigenen `main` — ohne auf den Upstream-Merge zu warten.
3. Fremde Upstream-PRs werden nur bei konkretem Bedarf übernommen, immer per
   Merge des PR-Heads, nie per Cherry-Pick, und immer mit Herkunft in der
   Merge-Message.
4. Upstream wird monatlich mit einem Zwei-Minuten-Check verfolgt; Merges laufen
   über einen Auffangbranch mit grünem Test-Gate.
5. Die Identitätsfrage (`go.mod`) bleibt bewusst offen und wird am 2026-10-15
   anhand eines festgelegten Triggers neu bewertet.
