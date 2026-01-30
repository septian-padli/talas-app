const pinoHttp = require('pino-http');
const crypto = require('crypto');
const logger = require('../utils/logger'); // Use our custom configured logger instance

const loggingMiddleware = pinoHttp({
  logger: logger,
  genReqId: function (req) {
    if (req.headers['x-request-id']) return req.headers['x-request-id'];
    return crypto.randomUUID();
  },
  // Custom serializers if needed, but pino-http defaults are usually good
  serializers: {
    req: (req) => {
        // Customize what request fields to log if standard pino-http is too verbose or missing something
        return {
            id: req.id,
            method: req.method,
            url: req.url,
            // headers: req.headers, // headers are redacted by logger config if needed, but usually kept minimal
            // remoteAddress: req.remoteAddress,
            // remotePort: req.remotePort,
        };
    },
    res: (res) => {
        return {
            statusCode: res.statusCode
        };
    },
  },
  // Ensure we log response time
  // pino-http logs request finished by default which includes responseTime
});

module.exports = loggingMiddleware;
