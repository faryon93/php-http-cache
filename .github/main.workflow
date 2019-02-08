workflow "Build Docker Image" {
  on = "push"
  resolves = ["Build Alpine"]
}

action "Build Alpine" {
  uses = "actions/docker/cli@aea64bb1b97c42fa69b90523667fef56b90d7cff"
  args = "build -t faryon93/php-http-cache:latest ."
}
