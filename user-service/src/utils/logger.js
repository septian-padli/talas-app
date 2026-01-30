const pino = require('pino');

const logger = pino({
  level: process.env.LOG_LEVEL || 'info',
  redact: {
    paths: [
      'req.headers.authorization',
      'req.body.password',
      'req.body.token',
      'req.body.access_token',
      'req.body.refresh_token',
      'password',
      'token',
      'refresh_token',
      'access_token'
    ],
    remove: true
  },
  // Timestamp formatting
  timestamp: pino.stdTimeFunctions.isoTime,
});

module.exports = logger;
