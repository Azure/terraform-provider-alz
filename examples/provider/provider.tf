provider "alz" {
  alz_library_references = [
    {
      path = "platform/alz"
      ref  = "2024.03.03"
    },
    {
      custom_url = "${path.root}/lib"
    }
  ]
}
