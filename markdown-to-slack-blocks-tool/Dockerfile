FROM node:18-slim

# Create app directory
WORKDIR /usr/src/app

RUN npm install @tryfabric/mack

# Bundle app source
COPY server.js ./

EXPOSE 8080
CMD [ "node", "server.js" ]
