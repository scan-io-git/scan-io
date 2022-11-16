resource "aws_instance" "ec2_dev" {
  ami                    = "ami-09a2a0f7d2db8baca"
  instance_type          = "t2.micro"
  key_name               = aws_key_pair.key_ec2_dev_access.key_name
  vpc_security_group_ids = [aws_security_group.sg_access_ec2_dev.id]

  tags = {
    Name = "scanio-dev"
  }
}

resource "aws_security_group" "sg_access_ec2_dev" {
  egress = [
    {
      cidr_blocks      = ["0.0.0.0/0", ]
      description      = ""
      from_port        = 0
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "-1"
      security_groups  = []
      self             = false
      to_port          = 0
    }
  ]
  ingress = [
    {
      cidr_blocks      = ["0.0.0.0/0", ]
      description      = ""
      from_port        = 22
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
      to_port          = 22
    }
  ]
}

resource "aws_key_pair" "key_ec2_dev_access" {
  key_name   = "b.key_scanio"
  public_key = file("~/.ssh/id_ed25519.pub")
}
