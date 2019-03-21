workflow "Run tests on push" {
  on = "push"
  resolves = ["test"]
}

action "test" {
  uses = "cedrickring/golang-action@1.1.1"
  args = "go test -v"
}
