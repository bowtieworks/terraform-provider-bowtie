resource "bowtie_dns" "example" {
  name = "example.com"
  servers = [{
    addr = "192.0.2.1"
  }]
  exclude = [{
    name = "wrong.example.com"
  }]
}