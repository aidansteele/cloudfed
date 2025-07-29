variable "issuer_url" {
  type        = string
  description = "comes from the idp terraform output"
}

variable "oidc_aud" {
  type = string
}

variable "oidc_sub" {
  type = string
}

resource "aws_iam_openid_connect_provider" "example" {
  url = var.issuer_url
  client_id_list = [var.oidc_aud]
}

resource "random_pet" "example" {}

resource "aws_iam_role" "example" {
  name = random_pet.example.id
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = "sts:AssumeRoleWithWebIdentity"
        Principal = {
          Federated = aws_iam_openid_connect_provider.example.arn
        }
        Condition = {
          StringEquals = {
            "${substr(var.issuer_url, 8, length(var.issuer_url)-8)}:sub" : var.oidc_sub
          }
        }
      },
    ]
  })
}

resource "aws_iam_role_policy" "example" {
  role = aws_iam_role.example.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "s3:ListAllMyBuckets"
        Resource = "*"
      },
    ]
  })
}

output "role_arn" {
  value = aws_iam_role.example.arn
}
