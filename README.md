# Snake (Go backend, không CC/PIN)

## Cấu trúc
```
snake-app/
├── main.go              ← backend Go (chỉ dùng standard library)
├── go.mod
├── data.json             ← tự tạo khi server chạy lần đầu (lưu username + điểm)
└── public/
    ├── snake.html
    ├── static/js/game.js
    ├── static/js/hardcore.js
    └── OSTS/*.mp3        ← tự thêm file nhạc của bạn vào đây (không bắt buộc)
```

## Chạy
```bash
cd snake-app
go run .
```
Mặc định chạy ở `http://localhost:8080`. Đổi cổng bằng biến môi trường `PORT`:
```bash
PORT=3000 go run .
```

Build ra file thực thi:
```bash
go build -o snake-server .
./snake-server
```

## Những gì đã bỏ / đã đổi so với bản cũ
- **Bỏ hẳn**: PIN, hệ thống CC Points, `claim`, cooldown 24h, `game_session_id`.
- **Giữ nguyên**: Hardcore Mode (toàn bộ hiệu ứng chaos trong `hardcore.js`), combo, gold apple, D-pad, touch control, theme toggle.
- **Đăng ký**: chỉ cần username (2-20 ký tự, chữ/số/`_`), **phân biệt hoa/thường** khi kiểm tra trùng ("Alice" và "alice" là 2 tài khoản khác nhau). Nếu trùng → hiện lỗi ngay dưới ô nhập, người chơi sửa tên và bấm START lại (không cần load lại trang).
- **Giao diện**: đã bỏ nút "← BACK" và branding "CHOCO HUB" ở thanh trên cùng, chỉ còn nút đổi theme.
- **Điểm**: tự động nộp lên server khi Game Over (không cần bấm nút gì), lưu **điểm cao nhất** theo từng mode.
- **Leaderboard**: 3 tab — Normal, Hardcore, và **Tổng hợp** (= điểm cao nhất Normal + điểm cao nhất Hardcore của mỗi người).

## API
| Method | Path | Body | Ghi chú |
|---|---|---|---|
| POST | `/api/register` | `{"username":"abc"}` | 409 nếu trùng tên |
| POST | `/api/submit-score` | `{"username":"abc","mode":"normal","score":12}` | mode: `normal`\|`hardcore` |
| GET | `/api/leaderboard` | — | trả `{"normal":[...],"hardcore":[...],"combined":[...]}` |

## Lưu ý
- Không có mật khẩu/bảo mật nào bảo vệ username — đúng như yêu cầu "đăng ký đơn giản". Ai gõ đúng tên đã tồn tại cũng nộp điểm được dưới tên đó (không có xác thực chủ sở hữu). Nếu sau này cần chống giả mạo tên, có thể thêm cookie/token khi đăng ký.
- `data.json` là lưu trữ đơn giản dạng file, phù hợp quy mô nhỏ/vừa. Nếu cần chịu tải lớn hơn thì chuyển sang SQLite/Postgres sau.
- File không có dependency ngoài nên `go build`/`go run` chạy được cả trên máy không có mạng (miễn có sẵn Go toolchain) — kể cả trên Termux.
# Snake-game-web
# Snake-game-web
