# 🌀 My BitTorrent Client  

Just trying to build my own **BitTorrent client** because why not?  
This is more of a learning project to understand how the BitTorrent protocol works.  

---

## 🔗 Links I’m Following  

- **Blog for Implementation:**  
  [How to Make Your Own BitTorrent Client by Allen Kim](https://allenkim67.github.io/programming/2016/05/04/how-to-make-your-own-bittorrent-client.html)  

- **Unofficial BitTorrent Documentation:**  
  [BitTorrent Specification](https://wiki.theory.org/BitTorrentSpecification#Bencoding)  

- **Tracker Communication Specs:**  
  [Connect, Announce, Scrape Protocol](https://www.bittorrent.org/beps/bep_0015.html)  

- **Testing Torrents (Free & Legal!):**  
  [WebTorrent Free Torrents](https://webtorrent.io/free-torrents)  

- **Other helpful links**
  - [https://blog.jse.li/posts/torrent/](https://blog.jse.li/posts/torrent/)


---

## 🛠️ Current Progress  

Here’s what I’m working on:  

- **Parsing `.torrent` files**: Figuring out how to extract metadata like trackers and file details.  
- **Talking to Trackers (UDP)**: Learning how to send `Connect`, and `Announce` requests and get peers.  
- **Connecting with Peers**: Setting up handshakes and managing connections.  
- **Downloading Files**: Downloading blocks/pieces and putting them together.  
- **Rebuilding Files**: Splitting the downloaded content into original files.  

---

## 🚀 How to Use  

1. Clone this repo:  
   ```bash
   git clone <repository-url>
   cd <repository-folder>
    ```

2. Install dependencies:  
   ```bash
    go get
    ```

2. Run:  
   ```bash
    go run cmd/mybittorrent/main.go <path-to-your-torrent-file>
    ```