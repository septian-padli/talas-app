/**
 * Server Entry Point
 * This file starts the Express server.
 * Separated from app.js to allow Supertest to import the app without starting the server.
 */
require('dotenv').config();
const app = require('./app');

const { connectRabbitMQ } = require('./utils/rabbitmq');

const PORT = process.env.PORT || 3001;

// Initialize RabbitMQ connection
connectRabbitMQ();

app.listen(PORT, () => {
    console.log(`🚀 User Service running on port ${PORT}`);
});
