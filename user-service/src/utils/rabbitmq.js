const amqp = require('amqplib');
const { randomUUID } = require('crypto');

let channel = null;
let connection = null;

const connectRabbitMQ = async () => {
  if (connection) return; // Prevent multiple connections

  try {
    const amqpServer = process.env.RABBITMQ_URL || 'amqp://user:password@localhost:5672';
    connection = await amqp.connect(amqpServer);
    channel = await connection.createChannel();
    
    // Assert Exchange 'talas.events' (Topic)
    await channel.assertExchange('talas.events', 'topic', { durable: true });
    
    console.log('✅ RabbitMQ Connected & Exchange Asserted');

    // Handle connection closure
    connection.on('close', () => {
      console.error('❌ RabbitMQ Connection Closed');
      connection = null;
      channel = null;
    });

    connection.on('error', (err) => {
        console.error('❌ RabbitMQ Connection Error:', err);
        connection = null;
        channel = null;
    });

  } catch (error) {
    console.error('❌ RabbitMQ Connection Failed:', error);
    // Don't throw, let app continue running without RabbitMQ
  }
};

const publishEvent = async (routingKey, data) => {
  if (!channel) {
    console.error('⚠️ RabbitMQ channel is not available. Event skipped.');
    return;
  }

  try {
    const exchange = 'talas.events';
    const payload = JSON.stringify(data);
    const buffer = Buffer.from(payload);
    
    channel.publish(exchange, routingKey, buffer);
    // console.log(`📡 Event Published: ${routingKey}`);
  } catch (error) {
    console.error('❌ Publish Event Error:', error);
  }
};

module.exports = {
  connectRabbitMQ,
  publishEvent
};
