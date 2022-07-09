import random
import requests
import time

PORTS = [
    '15011',
    '15012',
    '15013',
]

VEHICLES = [
    'bike',
    'scooter',
    'car',
]

if __name__ == "__main__":
    print(f"starting load generator")
    time.sleep(3)
    while True:
        port = PORTS[random.randint(0, len(PORTS) - 1)]
        vehicle = VEHICLES[random.randint(0, len(VEHICLES) - 1)]
        print(f"requesting {vehicle} from {port}")
        resp = requests.get(f'http://localhost:{port}/{vehicle}')
        print(f"received {resp}")
        time.sleep(random.uniform(0.2, 0.4))
