data "aws_caller_identity" "current" {}

resource "aws_lambda_function" "idp" {
  function_name    = "cloudfed-idp"
  role             = aws_iam_role.iam_for_lambda.arn
  memory_size      = 1024
  timeout          = 15
  handler          = "yolo"
  runtime          = "provided.al2023"
  filename         = "lambda.zip"
  source_code_hash = data.archive_file.lambda.output_base64sha256

  environment {
    variables = {
      KEY_ID = aws_kms_key.oidc.id
    }
  }
}

resource "aws_lambda_function_url" "idp" {
  function_name = aws_lambda_function.idp.function_name
  authorization_type = "NONE"
  invoke_mode = "RESPONSE_STREAM"
}

output "issuer_base_url" {
  # trim the trailing slash
  value = substr(aws_lambda_function_url.idp.function_url, 0, length(aws_lambda_function_url.idp.function_url) - 1)
}

resource "null_resource" "lambda_build" {
  provisioner "local-exec" {
    command = "GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go -C idp build -o bootstrap"
  }

  triggers = {
    src = filebase64sha256("idp/idp.go")
  }
}

data "archive_file" "lambda" {
  depends_on = [null_resource.lambda_build]
  type        = "zip"
  source_file = "idp/bootstrap"
  output_path = "lambda.zip"
}

data "aws_iam_policy_document" "assume_role" {
  statement {
    effect = "Allow"

    principals {
      type = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

data "aws_iam_policy_document" "permissions" {
  statement {
    effect = "Allow"

    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]

    resources = ["arn:aws:logs:*:*:*"]
  }

  statement {
    effect = "Allow"

    actions = [
      "kms:GetPublicKey",
    ]

    resources = [aws_kms_key.oidc.arn]
  }
}

resource "aws_iam_role_policy" "lambda_logging" {
  role   = aws_iam_role.iam_for_lambda.id
  name   = "lambda_logging"
  policy = data.aws_iam_policy_document.permissions.json
}

resource "aws_iam_role" "iam_for_lambda" {
  name               = "iam_for_lambda"
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}

resource "aws_kms_key" "oidc" {
  multi_region             = true
  customer_master_key_spec = "RSA_2048"
  key_usage                = "SIGN_VERIFY"
  description              = "cloudfed"

  policy = jsonencode({
    Version = "2012-10-17"
    Id      = "root-policy"
    Statement = [
      {
        Sid      = "Delegate IAM"
        Effect   = "Allow"
        Action   = "kms:*"
        Resource = "*"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
      },
    ]
  })
}

output "key_id" {
  value = aws_kms_key.oidc.id
}