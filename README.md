# simple-log-storage

Distributed log storage that is using Serf, Raft, gRPC, Zap, Casbin, Testify and more 

This project is simply to get more experience around the distributed systems and how to create one.

specificities:
- Client side load balancing using **gRPC Resolver/Picker** (Round robin with atomicity to avoid to overload one 
  destination server in case of high number of simultaneous requests)
- Command line using **Cobra**
- Configuration management using **Viper**
- Testify for testing


