const amqp = require('amqplib');

let channel = null;

const connectRabbitMQ = async () => {
  try {
    const amqpServer = process.env.RABBITMQ_URL || 'amqp://user:password@localhost:5672';
    const connection = await amqp.connect(amqpServer);
    channel = await connection.createChannel();
    
    // Assert Exchange 'user_events' (Topic)
    await channel.assertExchange('user_events', 'topic', { durable: true });
    
    console.log('✅ RabbitMQ Connected & Exchange Asserted');
  } catch (error) {
    console.error('❌ RabbitMQ Connection Error:', error);
    // Retry logic could be added here
  }
};

const publishEvent = async (routingKey, data) => {
  if (!channel) {
    console.error('⚠️ RabbitMQ channel is not available. Event skipped.');
    return;
  }

  try {
    const exchange = 'user_events';
    const buffer = Buffer.from(JSON.stringify(data));
    
    channel.publish(exchange, routingKey, buffer);
    console.log(`📡 Event Published: ${routingKey}`);
  } catch (error) {
    console.error('❌ Publish Event Error:', error);
  }
};

// Initialize connection on load
connectRabbitMQ();

module.exports = {
  connectRabbitMQ,
  publishEvent
};
