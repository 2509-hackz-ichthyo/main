locals {
  bucket_name = "hackz-ichthyo-bucket"
}

resource "aws_s3_bucket" "hackz_ichthyo_bucket" {
  bucket = local.bucket_name
}

resource "aws_s3_object" "hackz_ichthyo_object" {
  bucket  = aws_s3_bucket.hackz_ichthyo_bucket.id
  key     = "../app/index.html"
  content = "Hello, Hackz Ichthyo!"
}

resource "aws_s3_bucket_policy" "policy" {
  depends_on = [
    aws_s3_bucket.hackz_ichthyo_bucket,
  ]
  bucket = aws_s3_bucket.hackz_ichthyo_bucket.id
  policy = data.aws_iam_policy_document.policy_document.json
}

data "aws_iam_policy_document" "policy_document" {
  statement {
    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }
    actions = ["s3:GetObject"]
    resources = [
      aws_s3_bucket.hackz_ichthyo_bucket.arn,
      "${aws_s3_bucket.hackz_ichthyo_bucket.arn}/*"
    ]
    condition {
      test     = "StringEquals"
      variable = "aws:SourceArn"
      values   = [aws_cloudfront_distribution.hackz_ichthyo_cfront.arn]
    }
  }
}

resource "aws_cloudfront_distribution" "hackz_ichthyo_cfront" {
  enabled             = true
  default_root_object = "../app/index.html"

  origin {
    origin_id                = aws_s3_bucket.hackz_ichthyo_bucket.id
    domain_name              = aws_s3_bucket.hackz_ichthyo_bucket.bucket_regional_domain_name
    origin_access_control_id = aws_cloudfront_origin_access_control.hackz_ichthyo_oac.id
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  default_cache_behavior {
    target_origin_id       = aws_s3_bucket.hackz_ichthyo_bucket.id
    viewer_protocol_policy = "redirect-to-https"
    cached_methods         = ["GET", "HEAD"]
    allowed_methods        = ["GET", "HEAD"]
    forwarded_values {
      query_string = false
      cookies {
        forward = "none"
      }
    }
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }
}

resource "aws_cloudfront_origin_access_control" "hackz_ichthyo_oac" {
  name                              = aws_s3_bucket.hackz_ichthyo_bucket.bucket_domain_name
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

output "cfront_domain_name" {
  value = aws_cloudfront_distribution.hackz_ichthyo_cfront.domain_name
}
