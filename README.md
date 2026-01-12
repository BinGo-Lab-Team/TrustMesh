# TrustMesh PoC

[English](./README.md) | [简体中文](./README_ZH.md)

## Introduction

TrustMesh is a decentralized consensus mechanism whose key distinction lies in using *emergence* to replace traditional PoW/PoS, while removing the requirement for block-to-block referencing. Its security is tied to node behavior, and the specific rules are defined according to the purpose of the network, without requiring nodes to maintain a global view. In theory, such a structure can achieve extremely high scalability.  
For more details, refer to the whitepaper: [TrustMesh Whitepaper v0.0.1](./docs/Whitepaper/v0.0.1/TrustMesh.md)

The PoC version 1.0 has been released and is implemented in Go. The code is distributed under the [MIT License](./LICENSE). Usage is described below.

## Experimental Data

The following table shows the convergence behavior of the PoC under typical parameters, which is one of the core characteristics of TrustMesh.

|  Round   | Winner Distribution |
| :------: | :-----------------: |
| 29423550 |         30          |
| 29423551 |         30          |
| 29423552 |        29/1         |
| 29423553 |       22/7/1        |
| 29423554 |         30          |
| 29423555 |        29/1         |
| 29423556 |         30          |
| 29423557 |        29/1         |
| 29423558 |         30          |
| 29423559 |       24/4/2        |
| 29423560 |        29/1         |
| 29423561 |       25/1/4        |
| 29423562 |      2/19/4/5       |
| 29423563 |       25/4/1        |
| 29423564 |       7/22/1        |
| 29423565 |         30          |
| 29423566 |        28/2         |
| 29423567 |       8/21/1        |
| 29423568 |        25/5         |
| 29423569 |        29/1         |
| 29423570 |        29/1         |
| 29423571 |        25/5         |
| 29423572 |        29/1         |
| 29423573 |        25/5         |
| 29423574 |        21/9         |
| 29423575 |        23/7         |
| 29423576 |         30          |
| 29423577 |        29/1         |
| 29423578 |       10/19/1       |
| 29423579 |         30          |
| 29423580 |        29/1         |

Below is the histogram of “number of supporters for the winning proposal”:

<p align="center">
<img src="media/winner_hist.svg" alt="performance graph">
</p>

1. **An average of 26.61 out of 30 nodes (88.7%) converged on the same proposal.**  
   This is achieved without global information, synchronous voting, or any chain-based structure.
2. **In every round, the winning proposal received more support than all other proposals combined.**  
   This shows strong amplification of early advantages and formation of dominant majorities.
3. **Even when multiple proposals coexist (e.g., 22/7/1 or 10/19/1), the final winner still maintains absolute dominance.**  
   Small groups only form local stable points and cannot block global convergence.

## Quick Start

### 1. Install Docker and Docker Compose

Please refer to the official Docker documentation:

- https://docs.docker.com/compose/install/

Verify installation:

```bash
docker --version
docker compose version
```

------

### 2. Download the Compose Generator Tool

You do **not** need to manually write `docker-compose.yml`; the tool generates it automatically.

**Linux**

```bash
wget https://github.com/BinGo-Lab-Team/TrustMesh/releases/download/PoC-1.0.0/Linux_amd64_MakeCompose
chmod +x Linux_amd64_MakeCompose
```

**Windows PowerShell**

```powershell
iwr https://github.com/BinGo-Lab-Team/TrustMesh/releases/download/PoC-1.0.0/Windows_amd64_MakeCompose.exe -OutFile MakeCompose.exe
```

------

### 3. Run the Configuration Tool

**Linux**

```bash
./Linux_amd64_MakeCompose
```

**Windows**

```powershell
.\MakeCompose.exe
```

The tool will prompt for configuration parameters.  
Reference (not used by the program):  
`.env`, `.env_bootstrap` — explanation-only files.

`docker-compose.yml` contains the actual configuration.

------

### 4. Start the Local TrustMesh Network

```bash
docker compose up -d
```

Check running nodes:

```bash
docker compose ps
```

Logs:

```bash
docker compose logs -f
```

------

### 5. Bootstrapping

Nodes initially do not know each other’s network locations. They must contact the bootstrap node on first startup to receive neighbor information.

If the database does not exist, it will be created automatically and a bootstrap request will be triggered.  
If the bootstrap node is not yet online, start it first and restart the other nodes afterward.  
To request again, delete `data.db` and restart the node.

------

### 6. Stop and Clean Up

```bash
docker compose down
docker compose down -v
```

------

### 7. Analysis

Winning block data is stored as JSON at:

```
volumes folder/node-x/block/xxx.json
```

**Linux Analyzer Tool**

```bash
wget https://github.com/BinGo-Lab-Team/TrustMesh/releases/download/PoC-1.0.0/Linux_amd64_Analyzer
chmod +x Linux_amd64_Analyzer
```

**Windows Analyzer Tool**

```powershell
iwr https://github.com/BinGo-Lab-Team/TrustMesh/releases/download/PoC-1.0.0/Windows_amd64_Analyzer.exe -OutFile Analyzer.exe
```

Run it and input the volumes path and round number.

## Additional Information

If you have any questions, please submit them on Issues or email me at yangzhixun-@outlook.com
