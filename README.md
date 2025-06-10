# EZModbus

Setup your server by editing the `config.json` file.

**The `server` section:**
This section handles the "Modbus TCP" part. It's about how other devices (clients) find and connect to the server over a network.

```JSON

  "server": {
    "address": "0.0.0.0",
    "port": 1502,
    "max_clients": 10,
    "timeout": 30,
    "max_retries": 3,
    "retry_delay": 5
  },
```

- `"address": "0.0.0.0"`: This is the IP address your server will listen on. `0.0.0.0` is a special address that means "listen for connections on all available network interfaces on this machine." For production, this is typical, but you would use a firewall to restrict which external IPs can actually connect to it.

- `"port": 1502`: This is the standard, registered network port for the Modbus protocol. Think of it like port 80 for web pages. All Modbus clients will try to connect on this port by default.

- `"max_clients": 10`: This defines how many Modbus clients (often called "Masters") can be connected to your server at the same time. In Modbus, one or more Masters poll a Slave (your server) for data. This setting prevents your server from being overwhelmed.

- `"timeout", "max_retries", "retry_delay"`: These are general reliability settings for your specific server application, allowing it to handle network hiccups gracefully upon startup.

The `modbus` section: The Protocol Logic
This section defines the "Modbus" data model itself. This is the heart of your virtual device, describing its identity and its "memory."

```JSON

  "modbus": {
    "unit_id": 1,
    "max_registers": 1000,
    "counter_address": 102,
    "update_interval": 1,
    "initial_data": [ ... ]
  }
```

- `"unit_id": 1`: This is a critical Modbus concept. It's the "Slave ID" or "Device Address." Before network cables were common, multiple physical devices might share a single serial cable. The Unit ID was used to address a specific device on that shared cable. In Modbus TCP, it acts as a logical address. Every request from a client includes a Unit ID, and your server will only respond if the ID in the request matches this value. This allows a single server to potentially simulate multiple devices, though your current code simulates just one.

- `"max_registers": 1000`: This allocates the "memory map" for your device. Modbus devices expose their data through four types of simple data tables. This setting defines how many slots are available in each of those tables (from address 0 to 999).

- `"initial_data": [ ... ]`: This is where you define the data inside your virtual device's memory tables. Modbus has four primary data types:

    - **Holding Registers ("type": "holding")**: This is the most common data type. It's a 16-bit number (0-65535) that can be read and written by a client. Think of it as a variable or a setting. In your example, a client can read that address 100 has a value of 2024 and can also send a command to change that value.

    - **Input Registers**: These are also 16-bit numbers, but they are read-only. You use these for values that the client should not be able to change, like a temperature sensor reading or a device's serial number.

    - **Coils**: These are single bits that can be read and written. They represent a simple on/off state, like a switch, a motor, or a valve.

    - **Discrete Inputs**: These are single bits that are read-only. They represent a status that the client cannot change, like a physical alarm sensor or a "door open" switch.

- `"counter_address": 102` and `"update_interval": 1`: These are custom features of your specific server program. You've created a special "live" data point. This tells your server to take the holding register at address 102 and automatically increment its value every 1 second. This is great for testing, as it simulates a device that has changing data.

**Configuration Examples**
Here are a few ways to set up this file for different purposes. (NOTE) `port: 502` is the default port for Modbus, that port requires priv esc on linux.

**Example 1: Simple Local Development**
This setup is for when you are testing on your own machine. It's configured to be easy to debug and not accessible from the outside network.

```JSON

{
  "server": {
    "address": "127.0.0.1",
    "port": 502,
    "max_clients": 2
  },
  "logging": {
    "level": "DEBUG",
    "console": true
  },
  "modbus": {
    "unit_id": 1,
    "max_registers": 100,
    "counter_address": 10,
    "update_interval": 2,
    "initial_data": [
      { "type": "coil", "address": 0, "value": 1 },
      { "type": "holding", "address": 0, "value": 1234 }
    ]
  }
}
```
**Example 2: Simulating a "Smart HVAC Unit"**
This shows how you'd configure the server to mimic a specific real-world device with a defined data map.

```JSON

{
  "server": {
    "address": "0.0.0.0",
    "port": 502,
    "max_clients": 20
  },
  "logging": {
    "level": "INFO",
    "console": true
  },
  "modbus": {
    "unit_id": 15,
    "max_registers": 200,
    "counter_address": 99,
    "update_interval": 1,
    "initial_data": [
      { "type": "coil", "address": 0, "value": 0 },             
      { "type": "coil", "address": 1, "value": 0 },             
      { "type": "discrete", "address": 0, "value": 1 },          
      { "type": "discrete", "address": 1, "value": 0 },         
      { "type": "holding", "address": 10, "value": 720 },        
      { "type": "input", "address": 20, "value": 715 },        
      { "type": "input", "address": 21, "value": 45 }         
    ]
  }
}
```

**Example 3: Enterprise Production Setup**
(Note): Can't use comments in JSON so dont put the comments in your config file

```JSON

{
  "server": {
    "address": "0.0.0.0",   
    "port": 502,
    "max_clients": 50,      
    "timeout": 60,          
    "max_retries": 10,
    "retry_delay": 15
  },
  "logging": {
    "level": "INFO",        
    "file": "/logs/modbus_server/modbus_server.jsonl",
    "max_size_mb": 500,     
    "console": false       
  },
  "modbus": {
    "unit_id": 101,          
    "max_registers": 20000,
    "counter_address": 1,
    "update_interval": 1,
    "initial_data": [
      // ======================================================================
      //   CONTROL COILS (Addresses 0-99) - Writable On/Off Switches
      // ======================================================================
      { "type": "coil", "address": 0, "value": 0 },       // Main Breaker Control (0=Open, 1=Close)
      { "type": "coil", "address": 1, "value": 0 },       // Backup Generator Control (0=Disable, 1=Enable)
      { "type": "coil", "address": 10, "value": 0 },      // Clear Latched Alarms (Write 1 to clear)

      // ======================================================================
      //   STATUS DISCRETE INPUTS (Addresses 100-199) - Read-Only On/Off Status
      // ======================================================================
      { "type": "discrete", "address": 100, "value": 0 }, // Main Breaker Status (0=Open, 1=Closed)
      { "type": "discrete", "address": 101, "value": 0 }, // Backup Generator Status (0=Stopped, 1=Running)
      { "type": "discrete", "address": 102, "value": 1 }, // Control Mode (0=Manual, 1=Remote/Modbus)
      { "type": "discrete", "address": 110, "value": 0 }, // High Temperature Alarm (0=OK, 1=Alarm)
      { "type": "discrete", "address": 111, "value": 0 }, // Over-voltage Fault (0=OK, 1=Fault)
      { "type": "discrete", "address": 112, "value": 0 }, // Under-voltage Fault (0=OK, 1=Fault)

      // ======================================================================
      //   CONFIGURATION HOLDING REGISTERS (Addresses 1000-1999) - Writable Settings
      // ======================================================================
      { "type": "holding", "address": 1000, "value": 4900 }, // Over-voltage Trip Point (Value in Volts * 10, e.g., 490.0V)
      { "type": "holding", "address": 1001, "value": 4700 }, // Under-voltage Trip Point (Value in Volts * 10, e.g., 470.0V)
      { "type": "holding", "address": 1002, "value": 85 },   // High Temperature Alarm Point (Celsius)
      { "type": "holding", "address": 1010, "value": 5000 }, // Over-current Trip Point (Amps * 100, e.g. 50.00A)

      // --- Writable Network Settings Example ---
      { "type": "holding", "address": 1100, "value": 192 },  // Static IP Octet 1
      { "type": "holding", "address": 1101, "value": 168 },  // Static IP Octet 2
      { "type": "holding", "address": 1102, "value": 1 },    // Static IP Octet 3
      { "type": "holding", "address": 1103, "value": 101 },  // Static IP Octet 4

      // ======================================================================
      //   MEASUREMENT INPUT REGISTERS (Addresses 2000-2999) - Read-Only Live Data
      // ======================================================================
      // --- Voltage (unit: Volts * 10) ---
      { "type": "input", "address": 2000, "value": 4801 },   // Voltage Phase A-B
      { "type": "input", "address": 2001, "value": 4805 },   // Voltage Phase B-C
      { "type": "input", "address": 2002, "value": 4799 },   // Voltage Phase C-A
      // --- Current (unit: Amps * 100) ---
      { "type": "input", "address": 2010, "value": 1550 },   // Current Phase A (e.g., 15.50A)
      { "type": "input", "address": 2011, "value": 1545 },   // Current Phase B
      { "type": "input", "address": 2012, "value": 1552 },   // Current Phase C
      // --- Power & Frequency ---
      { "type": "input", "address": 2020, "value": 99 },     // Power Factor (Value * 100, e.g., 0.99)
      { "type": "input", "address": 2021, "value": 450 },    // Active Power (kW)
      { "type": "input", "address": 2030, "value": 6001 },   // Frequency (Hz * 100, e.g., 60.01Hz)

      // ======================================================================
      //   IDENTIFICATION & STATS INPUT REGISTERS (Addresses 3000-3999) - Read-Only
      // ======================================================================
      { "type": "input", "address": 3000, "value": 2 },      // Firmware Version Major
      { "type": "input", "address": 3001, "value": 14 },     // Firmware Version Minor
      // --- Serial Number stored as ASCII bytes (2 chars per register) ---
      { "type": "input", "address": 3002, "value": 21072},   // "PM" (P=80, M=77 -> 80*256 + 77 = 20557 is wrong, should be bitshifted: 80<<8 | 77 = 20557) - let's use a simpler value for clarity if ASCII is too complex. Let's use decimal values.
      { "type": "input", "address": 3002, "value": 8077 },   // ASCII for "PM" as a simple decimal
      { "type": "input", "address": 3003, "value": 6785 },   // ASCII for "CU" as a simple decimal
      { "type": "input", "address": 3004, "value": 4549 },   // ASCII for "-1" as a simple decimal
      { "type": "input", "address": 3005, "value": 4848 },   // ASCII for "00" as a simple decimal
      { "type": "input", "address": 3006, "value": 5651 },   // ASCII for "83" as a simple decimal
      // --- Uptime and Counters ---
      { "type": "input", "address": 3020, "value": 0 },      // Your server's live counter (seconds) will update this address
      { "type": "input", "address": 3021, "value": 1345 }    // Total Breaker Operations
    ]
  }
}
```
