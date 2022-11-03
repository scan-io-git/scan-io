resource "aws_ecr_repository" "scanio_repository" {
  name = "scanio"
  force_delete = true

  image_scanning_configuration {
    scan_on_push = false
  }
}
