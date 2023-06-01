FROM node:18

# Create app directory
WORKDIR /usr/src/app

RUN npm install @tryfabric/mack

RUN npm ci --omit=dev

# Bundle app source
COPY server.js ./

EXPOSE 8080
CMD [ "node", "server.js" ]
