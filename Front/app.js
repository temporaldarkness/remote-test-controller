const ws = new WebSocket("ws://localhost:8080/ws");

let timerInterval = null;
let startTime = null;
let isRunning = false;
let isPaused = false;

const btnStart = document.getElementById('btn-start');
const btnPause = document.getElementById('btn-pause');
const btnStop = document.getElementById('btn-stop');
const startTimeEl = document.getElementById('start-time');

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
    stopTimer();
    updateTimer();
    timerInterval = setInterval(updateTimer, 1000);
}

function stopTimer() {
    clearInterval(timerInterval);
    timerInterval = null;
    startTimeEl.textContent = '00:00:00';
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
    if (data.startTime) {
        startTime = new Date(data.startTime);
    } else {
        startTime = null;
    }
    if (isRunning && !isPaused && startTime) {
        startTimer();
    } else if (isRunning && isPaused) {
        stopTimer();
        // Можно оставить последнее время на паузе
        updateTimer();
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