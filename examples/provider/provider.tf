provider "alz" {
  library_references = [
    {
      path = "platform/alz"
      ref  = "2024.07.5"
    },
    {
      custom_url = "${path.root}/lib"
    }
  ]
}
