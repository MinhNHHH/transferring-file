import { WebSocket } from "ws";

const PORT : any = 3001
const wsServer = new WebSocket.Server({ port: PORT });

const list_room: Array<any> = [];

interface Room {
  roomId: string;
  connections: Array<WebSocket>;
}

interface message {
  event: string;
  message: any;
}

class Room {
  constructor(roomId: any) {
    this.roomId = roomId;
    this.connections = [];
  }

  addConnection(connection: WebSocket) {
    if (!this.connections.includes(connection)) {
      return this.connections.push(connection);
    }
  }

  boardcastException(msg: message, connection: WebSocket) {
    this.connections.forEach(function (client) {
      if (client !== connection && client.readyState === WebSocket.OPEN) {
        client.send(
          JSON.stringify({
            ...msg
          })
        );
      }
    });
  }

  handleDeleteConnection(connection: any) {
    this.connections = this.connections.filter((connection) => {
      return connection.readyState !== 3;
    });
  }
}

wsServer.on("connection", (ws: WebSocket, request) => {
  // check room existed and create room and add connection
  let room: Room = list_room.find((r) => r.roomId === request.url);
  if (!room) {
    room = new Room(request.url);
    list_room.push(room);
  }
  // If room existed add another connection
  room.addConnection(ws);
  ws.on("message", (message: Buffer) => {
    const msg = JSON.parse(message.toString("utf-8"));
    // handle message
    room.boardcastException(msg, ws);
  });
  ws.on("close", () => {
    room.handleDeleteConnection(ws);
    console.log("Client has disconnected.");
  });
});
