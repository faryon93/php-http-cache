workflow "Build" {
  on = "push"
  resolves = ["Publish on Docker Registry"]
}

action "Build Alpine" {
  uses = "actions/docker/cli@aea64bb1b97c42fa69b90523667fef56b90d7cff"
  args = "build -t faryon93/php-http-cache:latest ."
}

action "Docker Registry Login" {
  uses = "actions/docker/login@master"
  needs = ["Build Alpine"]
  secrets = ["DOCKER_USERNAME", "DOCKER_PASSWORD"]
}

action "Publish on Docker Registry" {
  uses = "actions/docker/cli@aea64bb1b97c42fa69b90523667fef56b90d7cff"
  args = "push faryon93/php-http-cache:latest"
  needs = ["Docker Registry Login"]
}
