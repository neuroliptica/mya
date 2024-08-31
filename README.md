# mya
Имиджборда в учебных целях. Фронт вдохновлен cutechan.

## Build
* Go >= 1.18
* C11 compiler
* make
* pkg-config
* pthread
* ffmpeg >= 4.1 libraries (libavcodec, libavutil, libavformat, libswscale)

```bash
$ sudo apt-get update
$ sudo apt-get install graphicsmagick-libmagick-dev-compat \
	build-essential \
	pkg-config \
	curl \
	libavcodec-dev \
	libavutil-dev \
	libavformat-dev \
	libswscale-dev
$ git clone https://github.com/neuroliptica/mya.git
$ cd mya
$ go build
```

## Start
```bash
$ ./mya
```
При первом запуске нужно будет создать хотя бы одну доску. Отправьте POST запрос на эндпоинт /api/create_board следующего вида:
```bash
$ curl -X POST http://localhost:3000/api/create_board \
-H 'Content-Type: application/json' \
-d '{"link":"b","name":"блинчики"}'
```
`link` отвечает за код доски, `name` за её длинное название. Оба поля должны быть уникальны. При успешном запросе в ответ получите что-то такое.
```json
{"id":1,"name":"блинчики","link":"b"}
```
Далее можете зайти в браузере на http://localhost:3000/b

## todo
1. На странице просмотра доски обрезать слишком длинные посты.
2. Очищать форму после успешно отправленного поста.
3. Добавить кнопки разметки в форму. Добавить спойлер.
4. Переписать весь css, избавиться от мусора.
