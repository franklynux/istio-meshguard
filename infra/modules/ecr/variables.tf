variable "name" {
  description = "Full repository name, e.g. istio-meshguard/orders"
  type        = string
}

variable "image_tag_mutability" {
  description = "IMMUTABLE stops an existing tag being overwritten"
  type        = string
  default     = "IMMUTABLE"
}

variable "max_image_count" {
  description = "How many images to keep before older ones expire"
  type        = number
  default     = 5
}