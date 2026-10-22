'use strict';

const path = require('path');
const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');
const { sendConfirmation } = require('./confirmations');

const PORT = 50053;

// In the repo the proto sits two levels up. The Docker image sets PROTO_PATH.
const PROTO_PATH =
  process.env.PROTO_PATH || path.join(__dirname, '..', '..', 'proto', 'notifications.proto');

// No codegen: the .proto file is parsed when the process starts.
const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
  keepCase: true, // keep snake_case field names (order_id), as written in the proto
  defaults: true, // missing fields arrive as "" / false, matching proto3 semantics
});
const notifications = grpc.loadPackageDefinition(packageDefinition).notifications;

function handleSendConfirmation(call, callback) {
  const { order_id: orderId, email } = call.request;
  const result = sendConfirmation(orderId, email);
  // Log the order id only, never the email address.
  console.log(`confirmation order=${orderId} sent=${result.sent}`);
  callback(null, { sent: result.sent });
}

function main() {
  const server = new grpc.Server();
  server.addService(notifications.Notifications.service, {
    SendConfirmation: handleSendConfirmation,
  });

  server.bindAsync(`0.0.0.0:${PORT}`, grpc.ServerCredentials.createInsecure(), (err) => {
    if (err) {
      console.error(`bind failed: ${err.message}`);
      process.exit(1);
    }
    console.log(`notifications listening on :${PORT}`);
  });

  // Kubernetes sends SIGTERM when stopping a pod: finish in-flight calls,
  // but force the exit after 5 seconds so a stuck call can't block shutdown.
  const shutdown = () => {
    console.log('shutting down');
    setTimeout(() => {
      server.forceShutdown();
      process.exit(1);
    }, 5000).unref();
    server.tryShutdown(() => process.exit(0));
  };
  process.on('SIGTERM', shutdown);
  process.on('SIGINT', shutdown);
}

main();