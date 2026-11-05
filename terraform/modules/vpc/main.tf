resource "aws_vpc" "cluster_vpc" {
  cidr_block           = var.cidr_block
  enable_dns_hostnames = true
  enable_dns_support   = true
  tags = {
    Name = var.vpc_name
  }
}


data "aws_availability_zones" "ready" {
  state = "available"
  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_subnet" "pub-subnet" {
  count             = 2
  cidr_block        = cidrsubnet(aws_vpc.cluster_vpc.cidr_block, 8, count.index)
  vpc_id            = aws_vpc.cluster_vpc.id
  availability_zone = slice(data.aws_availability_zones.ready.names, 0, 2)[count.index]

  map_public_ip_on_launch = true

  tags = {
    Name                     = "public-subnet-${count.index}"
    Tier                     = "public"
    "kubernetes.io/role/elb" = "1"
  }
}

resource "aws_subnet" "priv-subnet" {
  count             = 2
  cidr_block        = cidrsubnet(aws_vpc.cluster_vpc.cidr_block, 4, count.index + 2)
  vpc_id            = aws_vpc.cluster_vpc.id
  availability_zone = slice(data.aws_availability_zones.ready.names, 0, 2)[count.index]

  map_public_ip_on_launch = false

  tags = {
    Name                              = "private-subnet-${count.index}"
    Tier                              = "private"
    Project                           = "Fleetform"
    "kubernetes.io/role/internal-elb" = "1"
  }
}

resource "aws_route_table" "pub-rt" {
  vpc_id = aws_vpc.cluster_vpc.id

  tags = {

    Name    = "Pub-rtb"
    Tier    = "public"
    Project = "Fleetform"

  }

}

resource "aws_route_table" "priv-rt" {
  vpc_id = aws_vpc.cluster_vpc.id

  tags = {

    Name    = "Priv-rtb"
    Tier    = "private"
    Project = "Fleetform"

  }
}

resource "aws_route" "priv-route" {
  route_table_id         = aws_route_table.priv-rt.id
  gateway_id             = aws_nat_gateway.nat-GW.id
  destination_cidr_block = "0.0.0.0/0"
}

resource "aws_internet_gateway" "IGW" {
  vpc_id = aws_vpc.cluster_vpc.id
}

resource "aws_route" "pub-route" {
  route_table_id         = aws_route_table.pub-rt.id
  gateway_id             = aws_internet_gateway.IGW.id
  destination_cidr_block = "0.0.0.0/0"
}

resource "aws_route_table_association" "pub_rtb-assoc" {
  count          = 2
  route_table_id = aws_route_table.pub-rt.id
  subnet_id      = aws_subnet.pub-subnet[count.index].id

}

resource "aws_eip" "nat-eip" {
  domain = "vpc"
}

resource "aws_nat_gateway" "nat-GW" {
  allocation_id = aws_eip.nat-eip.id
  subnet_id     = aws_subnet.pub-subnet[0].id
}

resource "aws_route_table_association" "priv-rtb-assoc" {
  count          = 2
  route_table_id = aws_route_table.priv-rt.id
  subnet_id      = aws_subnet.priv-subnet[count.index].id

}

