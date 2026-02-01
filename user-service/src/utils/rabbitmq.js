const amqp = require('amqplib');
const { randomUUID } = require('crypto');


let channel = null;
let connection = null;
let isConnecting = false;
let reconnectTimeout = null;
const eventQueue = [];
const RECONNECT_INTERVAL = 5000; // ms


const connectRabbitMQ = async () => {
  if (isConnecting || connection) return;
  isConnecting = true;
  try {
    const amqpServer = process.env.RABBITMQ_URL || 'amqp://user:password@localhost:5672';
    connection = await amqp.connect(amqpServer);
    channel = await connection.createChannel();
    await channel.assertExchange('talas.events', 'topic', { durable: true });
    console.log('✅ RabbitMQ Connected & Exchange Asserted');

    // Flush any queued events
    flushEventQueue();

    connection.on('close', () => {
      console.error('❌ RabbitMQ Connection Closed');
      cleanupAndReconnect();
    });
    connection.on('error', (err) => {
      console.error('❌ RabbitMQ Connection Error:', err);
      cleanupAndReconnect();
    });
    channel.on('close', () => {
      console.error('❌ RabbitMQ Channel Closed');
      cleanupAndReconnect();
    });
    channel.on('error', (err) => {
      console.error('❌ RabbitMQ Channel Error:', err);
      cleanupAndReconnect();
    });
  } catch (error) {
    console.error('❌ RabbitMQ Connection Failed:', error);
    scheduleReconnect();
  } finally {
    isConnecting = false;
  }
};

function cleanupAndReconnect() {
  if (connection) {
    try { connection.removeAllListeners(); } catch {}
    try { connection.close(); } catch {}
  }
  if (channel) {
    try { channel.removeAllListeners(); } catch {}
    try { channel.close(); } catch {}
  }
  connection = null;
  channel = null;
  scheduleReconnect();
}

function scheduleReconnect() {
  if (reconnectTimeout) return;
  reconnectTimeout = setTimeout(() => {
    reconnectTimeout = null;
    connectRabbitMQ();
  }, RECONNECT_INTERVAL);
}

function flushEventQueue() {
  while (channel && eventQueue.length > 0) {
    const { exchange, routingKey, buffer } = eventQueue.shift();
    try {
      channel.publish(exchange, routingKey, buffer);
    } catch (err) {
      console.error('❌ Failed to flush queued event:', err);
      // If failed, re-queue and break
      eventQueue.unshift({ exchange, routingKey, buffer });
      break;
    }
  }
}


const publishEvent = async (routingKey, data) => {
  const exchange = 'talas.events';
  const payload = JSON.stringify(data);
  const buffer = Buffer.from(payload);

  if (!channel) {
    console.error('⚠️ RabbitMQ channel is not available. Event queued.');
    eventQueue.push({ exchange, routingKey, buffer });
    connectRabbitMQ(); // Try to reconnect if not already
    return;
  }

  try {
    channel.publish(exchange, routingKey, buffer);
    // console.log(`📡 Event Published: ${routingKey}`);
  } catch (error) {
    console.error('❌ Publish Event Error:', error);
    // On error, queue event for retry
    eventQueue.push({ exchange, routingKey, buffer });
    cleanupAndReconnect();
  }
};

module.exports = {
  connectRabbitMQ,
  publishEvent
};
