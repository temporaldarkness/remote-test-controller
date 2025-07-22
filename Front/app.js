const ws = new WebSocket("ws://localhost:8080/ws");

let timerInterval = null;
let dataPollInterval = null; // Новая переменная для интервала опроса данных
let startTime = null;
let isRunning = false;
let isPaused = false;

const btnStart = document.getElementById('btn-start');
const btnPause = document.getElementById('btn-pause');
const btnStop = document.getElementById('btn-stop');
const startTimeEl = document.getElementById('start-time');
const tempEl = document.getElementById('temperature');
const rpmEl = document.getElementById('rpm');




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
            ws.send(JSON.stringify({ action: "ping" }));
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

ws.onopen = function() {
    ws.send(JSON.stringify({ action: "status" }));
};

ws.onmessage = function(event) {
    const data = JSON.parse(event.data);
    isRunning = data.running;
    isPaused = data.paused;
    tempEl.textContent = data.temperature.toFixed(1);
    rpmEl.textContent = data.rpm;
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
    ws.send(JSON.stringify({ action: "start" }));
};
btnPause.onclick = function() {
    ws.send(JSON.stringify({ action: "pause" }));
};
btnStop.onclick = function() {
    ws.send(JSON.stringify({ action: "stop" }));
};

// Инициализация
startTimeEl.textContent = '00:00:00';
updateButtons();