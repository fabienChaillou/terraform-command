## USAGE Surcharge cmd custom
---

## 1. Charger une liste de commandes custom dans une registry

Tu as deux options selon le cas d'usage :

**Option A — Repartir de zéro** (registry vide, que tes commandes) :
```go
import "github.com/fabienChaillou/terraform-commander/terraform"

r := terraform.NewRegistryOf[terraform.Command]()
r.Register("init",    &MyInitCommand{})
r.Register("deploy",  &MyDeployCommand{})
// etc.
```

**Option B — Partir de la registry existante et surcharger** (garde les 11 commandes built-in, tu remplaces juste `init`) :
```go
import "github.com/fabienChaillou/terraform-commander/terraform/command"

r := command.NewRegistry()          // init, plan, apply, destroy, workspace, state…
r.Register("init", &MyInitCommand{}) // écrase uniquement "init"
```

---

## 2. Définir tes propres interfaces de commande

### Ton exemple exact : `init` qui valide uniquement `-upgrade` et injecte `-no-color` par défaut

```go
package mycommand

import "github.com/fabienChaillou/terraform-commander/terraform"

// MyInitCommand implémente terraform.Command (Validate + BuildArgs + Help).
type MyInitCommand struct{}

func (c *MyInitCommand) Help() string {
    return `Usage: terraform init [options]

Options:
  -upgrade    Upgrade providers and modules to latest allowed.
  -no-color   Injected automatically — cannot be disabled.
`
}

// Validate n'autorise QUE le flag -upgrade (et -no-color géré en interne).
func (c *MyInitCommand) Validate(req *terraform.Request) []terraform.ValidationError {
    allowed := map[string]bool{
        "upgrade": true,
    }
    var errs []terraform.ValidationError
    for k := range req.Args {
        if !allowed[k] {
            errs = append(errs, terraform.ValidationError{
                Field:   k,
                Message: "flag non autorisé pour myinit: -" + k,
            })
        }
    }
    return errs
}

// BuildArgs construit les args CLI en injectant toujours -no-color.
func (c *MyInitCommand) BuildArgs(req *terraform.Request) []string {
    args := []string{"init", "-no-color"} // toujours présent

    if v, ok := req.Args["upgrade"]; ok && isTruthy(v) {
        args = append(args, "-upgrade")
    }
    return args
}

func isTruthy(v interface{}) bool {
    switch b := v.(type) {
    case bool:   return b
    case string: return b == "true" || b == "1"
    }
    return false
}
```

### Aller plus loin — une interface custom plus spécifique

La registry est générique (`Registry[C Command]`), donc tu peux contraindre C à **ta propre interface** qui étend `terraform.Command` :

```go
// TFCommand étend Command avec un Timeout propre à ton domaine.
type TFCommand interface {
    terraform.Command
    Timeout() time.Duration
}

// MyInitCommand doit aussi implémenter Timeout().
func (c *MyInitCommand) Timeout() time.Duration { return 5 * time.Minute }

// Registry typée sur TFCommand — seules les commandes avec Timeout() peuvent être enregistrées.
r := terraform.NewRegistryOf[TFCommand]()
r.Register("init", &MyInitCommand{})

cmd, ok := r.Lookup("init")
if ok {
    fmt.Println(cmd.Timeout()) // disponible sans cast !
}
```

---

### Usage complet bout en bout

```go
r := command.NewRegistry()
r.Register("init", &MyInitCommand{}) // override

cmd, ok := r.Lookup("init")
if !ok {
    log.Fatal("commande introuvable")
}

req := &terraform.Request{
    Action: "init",
    Args:   map[string]interface{}{"upgrade": true},
}

if errs := cmd.Validate(req); len(errs) > 0 {
    fmt.Println(cmd.Help())
    // → affiche les erreurs de validation
}

cliArgs := cmd.BuildArgs(req)
// → ["init", "-no-color", "-upgrade"]
```

---

**Point clé** : `unknownFlagErrors` et `isTruthy` sont privées au package `command`, donc tu dois les réimplémenter dans ton propre package (comme ci-dessus). Tout le reste — `Registry`, `Command`, `Validator`, `ArgBuilder`, `HelpProvider`, `Request`, `ValidationError` — est exporté et utilisable directement.
