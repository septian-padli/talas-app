const amqp = require('amqplib');

// Sesuaikan kredensial dengan docker-compose
const RABBIT_URL = 'amqp://user:password@localhost:5672';

async function testConnection() {
  try {
    console.log("⏳ Mencoba menghubungkan ke RabbitMQ...");

    // 1. Connect
    const connection = await amqp.connect(RABBIT_URL);
    console.log("✅ KONEKSI BERHASIL: Terhubung ke RabbitMQ!");

    // 2. Create Channel
    const channel = await connection.createChannel();
    console.log("✅ CHANNEL BERHASIL: Jalur komunikasi siap.");

    // 3. Test Kirim Pesan ke Exchange Default
    const queue = 'test_queue';
    const msg = 'Halo RabbitMQ!';

    await channel.assertQueue(queue, { durable: false });
    channel.sendToQueue(queue, Buffer.from(msg));
    console.log(`✅ KIRIM PESAN: '${msg}' berhasil dikirim ke antrean '${queue}'`);

    // 4. Tutup
    setTimeout(() => {
      connection.close();
      process.exit(0);
    }, 500);

  } catch (error) {
    console.error("❌ KONEKSI GAGAL:", error.message);
    process.exit(1);
  }
}

testConnection();