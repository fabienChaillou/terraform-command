terraform {
  required_version = ">= 1.0"

  required_providers {
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.2"
    }
  }
}

# Génère un entier aléatoire entre 1 et 1000
resource "random_integer" "nombre" {
  min = 1
  max = 1000

  # Force la régénération à chaque apply (optionnel)
  keepers = {
    timestamp = timestamp()
  }
}

# Affiche le nombre aléatoire dans le shell via local-exec
resource "null_resource" "afficher_nombre" {
  triggers = {
    nombre = random_integer.nombre.result
  }

  provisioner "local-exec" {
    command = "echo 'Nombre aleatoire genere : ${random_integer.nombre.result}'"
  }
}

# Sortie pour afficher également le nombre via `terraform output`
output "nombre_aleatoire" {
  description = "Le nombre aléatoire généré"
  value       = random_integer.nombre.result
}
