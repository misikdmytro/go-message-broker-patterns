# What Happens When the Message Broker Dies?

This repo contains example code from my Medium post [**"What Happens When the Message Broker Dies?"**](https://medium.com/@dmytro.misik/what-happens-when-the-message-broker-dies-e5b9b8550b33).

It's a hands-on guide for making your service more resilient when Kafka, RabbitMQ, or any other broker decides to take a break. Each pattern in the article is backed by runnable Go code:

## 🧪 Included Patterns

- 🧱 Outbox Pattern  
- 📝 Persisted Command Pattern  
- 📥 Local Disk Queue (with retries & backoff)  
- 💡 Fallback to a Different Broker (Kafka ➝ RabbitMQ)

Each folder contains minimal, focused examples to help you understand and adapt these techniques in your own services.

## 📖 Read the article

👉 [What Happens When the Message Broker Dies?](https://medium.com/@dmytro.misik/what-happens-when-the-message-broker-dies-e5b9b8550b33)

## 🛠 Requirements

- Go 1.20+
- Docker (for running Kafka/RabbitMQ locally)

---

Feel free to fork, adapt, and break things (preferably not in production).
