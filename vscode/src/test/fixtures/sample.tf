resource "null_resource" "example" {
  triggers = {
    always = timestamp()
  }
}
