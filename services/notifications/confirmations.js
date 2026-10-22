'use strict'; // ECMAScript mode: turns silent JavaScript mistakes, like assigning to an undeclared variable, into errors.

// In-memory record of confirmations "sent". No real email goes out.
const MAX_STORED = 1000;
const confirmations = [];

function isNonEmptyString(value) {
  return typeof value === 'string' && value.trim() !== '';
}

function sendConfirmation(orderId, email) {
  if (!isNonEmptyString(orderId) || !isNonEmptyString(email)) {
    return { sent: false };
  }
  confirmations.push({ orderId, email });
  if (confirmations.length > MAX_STORED) {
    confirmations.shift(); // drop the oldest
  }
  return { sent: true };
}

function getConfirmations() {
  return [...confirmations]; // a copy, so callers can't mutate the internal state
}

function resetConfirmations() {
  confirmations.length = 0;
}

module.exports = { sendConfirmation, getConfirmations, resetConfirmations, MAX_STORED };