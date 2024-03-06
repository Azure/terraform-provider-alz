provider "alz" {
  alz_lib_ref = "platform/alz@v2024.03.00" # using a specific release from the ALZ platform library
  lib_urls = [
    "${path.root}/lib",                                     # local library
    "github.com/MyOrg/MyRepo//some/dir?ref=v1.1.0&depth=1", # checking out a specific version
  ]
}
