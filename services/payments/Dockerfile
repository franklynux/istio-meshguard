# syntax=docker/dockerfile:1
FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

# Install dependencies first so this layer is cached until requirements change.
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy only what the service needs at runtime (no tests, no venv).
COPY charge.py server.py payments_pb2.py payments_pb2_grpc.py ./

# Numeric UID so Kubernetes can verify runAsNonRoot.
RUN useradd --system --uid 10001 --no-create-home appuser
USER 10001

EXPOSE 50052
CMD ["python", "server.py"]