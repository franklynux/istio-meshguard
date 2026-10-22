'use strict';

const {
  sendConfirmation,
  getConfirmations,
  resetConfirmations,
  MAX_STORED,
} = require('./confirmations');

// Module-level state is shared across tests in a file, so start each test clean.
beforeEach(() => {
  resetConfirmations();
});

test('sendConfirmation records a sent confirmation', () => {
  const result = sendConfirmation('o1', 'frank@example.com');
  expect(result.sent).toBe(true);
  expect(getConfirmations()).toContainEqual({ orderId: 'o1', email: 'frank@example.com' });
});

test.each([
  ['missing order id', '', 'frank@example.com'],
  ['missing email', 'o2', '   '],
])('sendConfirmation refuses a %s and records nothing', (_label, orderId, email) => {
  const result = sendConfirmation(orderId, email);
  expect(result.sent).toBe(false);
  expect(getConfirmations()).toHaveLength(0);
});

test('keeps only the most recent MAX_STORED confirmations', () => {
  for (let i = 0; i < MAX_STORED + 5; i++) {
    sendConfirmation(`o${i}`, 'frank@example.com');
  }
  const stored = getConfirmations();
  expect(stored).toHaveLength(MAX_STORED);
  expect(stored[0].orderId).toBe('o5'); // the 5 oldest were dropped
});