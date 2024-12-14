import socket
import json
import struct
import threading
import time
import random
import base64

class TCPClient:
    def __init__(self, host, port, client_id, paths):
        self.host = host
        self.port = port
        self.client_id = client_id
        self.paths = paths
        self.conn = None
        self.buffer = b""

    def connect(self):
        self.conn = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.conn.connect((self.host, self.port))
        print(f"Connected to {self.host}:{self.port}")

    def register(self):
        request = {
            "request_id": random.randint(1, 1000000),
            "method": "POST",
            "path": "/register",
            "headers": {"Content-Type": "application/json"},
            "body": base64.b64encode(json.dumps({
                "client_id": self.client_id,
                "paths": self.paths
            }).encode()).decode()  # Encode to base64, then to string
        }

        msg = {
            "sub": "REG",
            "msg": base64.b64encode(json.dumps(request).encode()).decode(),  # Encode to base64, then to string
            "request": request["request_id"]
        }

        self.send_message(msg)
        print("Registration message sent")

    def send_message(self, msg):
        serialized = json.dumps(msg).encode()
        length = struct.pack('>I', len(serialized))
        self.conn.sendall(length + serialized)
        print(f"Sent message: {msg}")

    def receive_message(self):
        while True:
            if len(self.buffer) >= 4:
                length = struct.unpack('>I', self.buffer[:4])[0]
                if len(self.buffer) >= 4 + length:
                    message = self.buffer[4:4+length]
                    self.buffer = self.buffer[4+length:]
                    return json.loads(message.decode())
            data = self.conn.recv(4096)
            if not data:
                return None
            self.buffer += data

    def handle_messages(self):
        while True:
            msg = self.receive_message()
            if msg is None:
                break
            print(f"Received message: {msg}")
            # Handle the message here

    def run(self):
        self.connect()
        self.register()
        threading.Thread(target=self.handle_messages, daemon=True).start()

        while True:
            time.sleep(1)

if __name__ == "__main__":
    client = TCPClient("localhost", 8081, "python_client", ["/stocks", "/weather"])
    client.run()