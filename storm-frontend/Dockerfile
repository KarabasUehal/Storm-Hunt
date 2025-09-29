FROM node:20-alpine AS build

WORKDIR /app

COPY package*.json ./

RUN npm install
RUN npm install @craco/craco --save-dev

COPY . .    

EXPOSE 3000

CMD ["npm", "start"]