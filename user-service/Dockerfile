# 1. Base Image ringan
FROM node:18-alpine

# 2. Set working directory
WORKDIR /app

RUN apk add --no-cache openssl

# 3. Copy package.json dulu (biar cache optimal)
COPY package*.json ./
COPY prisma ./prisma/

# 4. Install dependencies
RUN npm install

# 5. Generate Prisma Client
RUN npx prisma generate

# 6. Copy seluruh source code
COPY . .

# 7. Expose port (Internal Container)
EXPOSE 3001

# 8. Command start
CMD ["npm", "run", "dev"]