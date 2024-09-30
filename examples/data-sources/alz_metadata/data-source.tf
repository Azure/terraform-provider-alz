data "alz_metadata" "example" {}

output "alz_library_refs" {
  description = "A list of the loaded ALZ Library references."
  value       = data.alz_metadata.example.alz_library_refs
}
