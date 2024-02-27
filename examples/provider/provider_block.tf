provider "alz" {
  lib_urls = [
    "${path.root}/lib",
    "github.com/MyOrg/MyRepo//some/dir?ref=v1.1.0&depth=1",
  ]
}
