let timerInterval = null;
let dataPollInterval = null; // Новая переменная для интервала опроса данных
let startTime = null;
let isRunning = false;
let isPaused = false;

const btnConn = document.getElementById('btn-connection');
const btnParam = document.getElementById('btn-parameter');
const btnStart = document.getElementById('btn-start');
const btnPause = document.getElementById('btn-pause');
const btnStop = document.getElementById('btn-stop');
const startTimeEl = document.getElementById('start-time');
const tempEl = document.getElementById('temperature');
const rpmEl = document.getElementById('rpm');
const powEl = document.getElementById('power');
const nameEl = document.getElementById('test-object');
const testEl = document.getElementById('test-number');
const connAddress = document.getElementById('connection-ip');
const connPort = document.getElementById('connection-port');
const connKey = document.getElementById('connection-key');
const paramTest = document.getElementById('parameter-test');
const paramCommand = document.getElementById('parameter-command');
const desc = document.getElementById('desc');
const presetSelect = document.getElementById('presetSelect');

var ws = null;
var address = null;
var port = null;
var key = null;
var test = "000";

const presetTable = [
	["Установка ПД-14", "127.0.0.1", "8080", "test_key"],
	["Установка ПД-16", "127.0.0.1", "8075", ""],
];

for (let i = 0; i < presetTable.length; i++) {
	let option = document.createElement('option');
	option.value = i;
	option.innerHTML = presetTable[i][0];
	presetSelect.appendChild(option);
}

function updateTimer() {
    if (startTime && isRunning && !isPaused) {
        const now = new Date();
        const diff = Math.floor((now - startTime) / 1000);
        const hours = Math.floor(diff / 3600);
        const minutes = Math.floor((diff % 3600) / 60);
        const seconds = diff % 60;
        startTimeEl.textContent = `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    }
}

function startTimer() {

    if (timerInterval) {
        return;
    }
    updateTimer();
    // Интервал для обновления отображаемого времени
    timerInterval = setInterval(updateTimer, 1000);
    // Интервал для запроса актуальных данных (температура, обороты) с сервера
    dataPollInterval = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ key: key, test: test, action: "ping" }));
        }
    }, 1000); // Опрашиваем сервер каждую секунду
}

function stopTimer() {
    clearInterval(timerInterval);
    clearInterval(dataPollInterval); // Также останавливаем опрос данных
    timerInterval = null;
    dataPollInterval = null;
    startTimeEl.textContent = '00:00:00';
    // Сбрасываем поля при полной остановке
    tempEl.textContent = '--';
    rpmEl.textContent = '--';
}

function updateButtons() {
    btnStart.disabled = isRunning && !isPaused;
    btnPause.disabled = !isRunning || isPaused;
    btnStop.disabled = !isRunning;
}

function wsOnOpen() {
    ws.send(JSON.stringify({ key: key, test: test, action: "status" }));
};

function wsOnClose() {
    desc.innerHTML = "Соединение закрыто";
}

function wsOnMessage(event) {
    desc.innerHTML = `${address}:${port}`;
    
    const data = JSON.parse(event.data);
    isRunning = data.running;
    isPaused = data.paused;
    
    // Хардкод, фронт должен будет генерировать таблицу исходя из посланных сервером полей
    tempEl.textContent = data.fields[0].value.toFixed(1);
    rpmEl.textContent = data.fields[1].value;
    powEl.textContent = data.fields[2].value;
    nameEl.textContent = data.name;
    testEl.textContent = data.test;
    if (data.startTime) {
        startTime = new Date(data.startTime);
    } else {
        startTime = null;
    }
    if (isRunning && !isPaused && startTime) {
        startTimer();
    } else if (isRunning && isPaused) {
        clearInterval(timerInterval);
        clearInterval(dataPollInterval);
        timerInterval = null;
        dataPollInterval = null;
    } else {
        stopTimer();
    }
    updateButtons();
};

btnStart.onclick = function() {
    if (!ws) return;
    ws.send(JSON.stringify({ key: key, test: test, action: "start" }));
};
btnPause.onclick = function() {
    if (!ws) return;
    ws.send(JSON.stringify({ key: key, test: test, action: "pause" }));
};
btnStop.onclick = function() {
    if (!ws) return;
    ws.send(JSON.stringify({ key: key, test: test, action: "stop" }));
};
btnConn.onclick = function() {
    address = connAddress.value || "";
    port = connPort.value || "";
    key = connKey.value || "";
    if (!address || !port) return;
    if (ws) ws.close();
    desc.innerHTML = "Соединение закрыто";
    ws = new WebSocket(`ws://${address}:${port}/ws`);
    ws.onopen = wsOnOpen;
    ws.onmessage = wsOnMessage;
    ws.onclose = wsOnClose;
};
btnParam.onclick = function() {
    command = paramCommand.value ?? "";
    if (paramTest.value)
        test = paramTest.value;
    
    if (command != "") {
        ws.send(JSON.stringify({ key: key, test: test, action: "command", command: command }));
    } else {
        ws.send(JSON.stringify({ key: key, test: test, action: "status" }));
    }
    paramTest.value = "";
    paramCommand.value = "";
};
presetSelect.addEventListener('change', function(){
	let index = presetSelect.value;
	
	if (index == -1) {
		connAddress.value = "";
		connPort.value = "";
		connKey.value = "";
		
		return;
	}
	
	connAddress.value = presetTable[index][1];
	connPort.value = presetTable[index][2];
	connKey.value = presetTable[index][3];
});

// Инициализация
startTimeEl.textContent = '00:00:00';
updateButtons();