# Design System: Telegram-backed Virtual Cloud Drive

**Project ID:** local-product-design

## 1. Visual Theme & Atmosphere

Giao dien can tao cam giac nhu mot private cloud control room: rieng tu, dang tin cay, nhanh, co tinh ky thuat nhung khong kho dung. Nguoi dung phai cam thay day la mot o cung cloud ao nghiem tuc, khong phai mot tool upload Telegram don gian.

Telegram la ha tang an phia sau. UI khong nen qua giong Telegram Messenger. Mau xanh Telegram co the xuat hien nhu mot dau hieu ket noi va hanh dong chinh, nhung ban sac san pham la cloud drive, sync, media va storage control.

Tinh cach thiet ke:
- Calm technical: ro rang, it on ao, uu tien trang thai he thong.
- Storage-first: file, folder, sync state va dung luong la trung tam.
- Media-ready: player can sau, gon, it gay nhieu.
- Privacy-aware: share link, proxy, session va domain phai tao cam giac an toan.
- Cross-device: mobile PWA phai nhe va nhanh; desktop dashboard phai manh va giau thong tin.

## 2. Color Palette & Roles

- **Deep Cloud Navy (#0B1220):** Nen chinh cho dashboard desktop, sidebar va cac man hinh dieu khien cloud. Mau nay tao cam giac sau, rieng tu va ha tang.
- **Panel Slate (#111827):** Nen cua card, queue drawer, settings panel va cac khoi ky thuat tren nen toi. Dung de tao lop noi nhe ma khong qua nang.
- **Telegram Signal Blue (#229ED9):** Mau hanh dong chinh cho Upload, Connect Telegram, Sync Now, Stream va navigation dang active. Day la cau noi thi giac voi Telegram nhung khong nen lam mau duy nhat cua he thong.
- **Syncing Cyan (#06B6D4):** Trang thai dang dong bo, dang stream, dang scan, dang upload hoac download. Dung cho progress glow, spinner, activity line.
- **Synced Green (#22C55E):** Trang thai thanh cong, da dong bo, online, link con hieu luc, daemon healthy.
- **Conflict Amber (#F59E0B):** Canh bao xung dot file, link sap het han, Telegram rate limit, proxy can cau hinh, thao tac can user chu y.
- **Revoked Red (#EF4444):** Delete, revoke share, transfer failed, session error va hanh dong pha huy.
- **Ice Surface (#F5F8FB):** Nen sang cho mobile PWA, public share page va cac man hinh can doc de dang.
- **Mist Border (#D8E1EA):** Duong vien nhe cho input, file row, card sang va divider.
- **Muted Metadata (#94A3B8):** Text phu nhu size, date, path, speed, message id va mo ta ngan.
- **Media Black (#05070A):** Nen rieng cho video/audio player, preview fullscreen va che do tap trung media.

## 3. Typography Rules

He thong chu nen mang cam giac hien dai, ky thuat va de doc o mat do file cao. Khong dung kieu chu qua trang tri. Tieu de can chac, gon, co tinh dieu khien. Body text can mem hon, ro rang, phu hop huong dan nguoi dung Viet Nam.

Quy tac:
- Tieu de man hinh dung weight manh, spacing gon, tao cam giac dashboard.
- Ten file dung medium weight de noi bat trong list/grid.
- Metadata nhu size, ngay tao, sync state phu dung co chu nho hon va mau Muted Metadata (#94A3B8).
- Nut hanh dong dung label ngan, dong tu ro nghia: "Tai len", "Dong bo ngay", "Tao link", "Thu hoi link".
- Loi va canh bao dung tieng Viet than thien, tranh thong bao ky thuat dai dong trong UI chinh.

## 4. Component Stylings

* **Primary Buttons:** Nut chinh co dang vien tron rong hoac pill-shaped, nen Telegram Signal Blue (#229ED9), chu trang, cam giac nhanh va tin cay. Dung cho Upload, Connect, Sync Now, Save.

* **Secondary Buttons:** Nen trong hoac nen slate nhe, vien Mist Border (#D8E1EA) tren light mode hoac vien trang mo tren dark mode. Dung cho Cancel, More options, Copy path.

* **Destructive Buttons:** Dung Revoked Red (#EF4444), nhung khong qua gay gat truoc khi user confirm. Cac hanh dong Delete, Revoke, Logout nen co confirm dialog ro rang.

* **Cards/Containers:** Card co goc bo mem vua phai, cam giac nhu storage module. Tren desktop dark mode, card dung Panel Slate (#111827) voi vien mo va shadow rat nhe. Tren mobile light mode, card dung Ice Surface (#F5F8FB) hoac trang gan nhu tinh khiet voi border Mist Border (#D8E1EA).

* **Sidebar:** Sidebar la trung tam dieu huong drive. No nen co nen Deep Cloud Navy (#0B1220), item active dung Telegram Signal Blue (#229ED9) hoac nen xanh mo. Sidebar hien thi Drive Root, Phone Uploads, Synced Folders, Shared Links, Media, Settings.

* **File Rows:** File row phai compact, de scan, co icon loai file, ten file, size, ngay, trang thai sync va action menu. Hover nen rat nhe. Trang thai file phai co badge ro: Da dong bo, Dang tai len, Chi tren may, Chi tren cloud, Xung dot.

* **File Cards:** Grid card uu tien thumbnail va ten file. Goc bo mem, overlay action chi hien khi hover/tap. Media card co icon play nho va duration neu co.

* **Upload Drop Zone:** Drop zone co vien net dut hoac vien phat sang nhe. Khi user keo file vao, Syncing Cyan (#06B6D4) tao cam giac dang kich hoat. Mobile upload can co CTA lon, ro: "Tai len tu dien thoai".

* **Sync Status Badge:** Badge co dang capsule nho. Synced dung xanh la, syncing dung cyan, conflict dung amber, failed dung red. Text ngan gon va de hieu.

* **Progress Bars:** Thanh progress mong, tron dau, co mau Syncing Cyan (#06B6D4) cho transfer dang chay va Synced Green (#22C55E) khi hoan tat. Nen co speed va remaining time khi can.

* **Inputs/Forms:** Input co vien yen tinh, focus ring xanh/cyan ro nhung khong choi. Form settings nen chia section ro rang, co helper text tieng Viet.

* **Modals:** Modal nhu control panel noi tren nen. Goc bo rong hon card thuong, shadow mem, backdrop toi vua du. Modal nguy hiem can highlight vung can chu y bang amber/red.

* **Public Share Page:** Trang public can sang, sach, it yeu to dashboard. File name, size, preview, download button va thong tin het han phai ro. Neu co password, form nhap mat khau can don gian va tao cam giac an toan.

* **Media Player:** Player dung Media Black (#05070A), control toi gian, uu tien noi dung. Loading/buffering dung Syncing Cyan (#06B6D4). Loi stream phai noi ro bang tieng Viet va co nut thu lai.

* **Settings Panels:** Settings nen chia thanh Telegram, Sync, Storage, Domain, Proxy, Advanced. Cac muc nang cao co the thu gon de tranh lam user moi bi ngop.

## 5. Layout Principles

Desktop dashboard nen dung cau truc ba vung:
- Sidebar dieu huong ben trai.
- Workspace file/folder o giua.
- Activity/context panel ben phai cho queue, sync state, preview ngan hoac metadata.

Mobile PWA nen dung cau truc mot cot:
- Header gon voi drive/account status.
- CTA upload lon.
- Recent files.
- Quick actions: Share, Search, Media.
- Bottom navigation neu can.

Public share page nen doc lap voi dashboard, tai nhanh, tap trung vao file va hanh dong download/stream.

Khoang trang can tao cam giac binh tinh va kiem soat, khong tao cam giac rong vo nghia. Man hinh co nhieu file can uu tien mat do thong tin, nhung van giu row height du de thao tac touch tren mobile.

## 6. Motion & Feedback

Chuyen dong nen co y nghia:
- Upload item xuat hien bang stagger nhe.
- Sync progress cap nhat muot nhung khong nhay.
- Drop zone phan hoi tuc thi khi drag file.
- Toast ngan gon, co action neu can.
- Media buffering co animation nhe, khong dung spinner qua lon.

Khong lam UI qua nhieu animation. San pham can tao cam giac dang tin cay hon la trinh dien.

## 7. Vietnamese UX Voice

Tone tieng Viet can than thien, ro rang, dung thuat ngu pho thong.

Nen dung:
- "Tai len" thay vi "Upload".
- "Dong bo" thay vi "Sync" trong UI chinh.
- "O dia ao" khi noi voi user pho thong.
- "Lien ket chia se" thay vi "Share token".
- "Thu hoi link" thay vi "Revoke".
- "May tinh nay" va "Dam may" de giai thich trang thai local/cloud.

Cac thuat ngu ky thuat nhu MTProto, WebDAV, proxy domain chi nen xuat hien trong Advanced hoac tai lieu.
