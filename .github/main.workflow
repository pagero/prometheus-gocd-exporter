workflow "Run tests on push" {
  on = "push"
  resolves = ["test"]
}

action "test" {
  uses = "cedrickring/golang-action@1.2.0"
  args = "make test"
}
