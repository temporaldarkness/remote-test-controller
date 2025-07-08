const ws = new WebSocket('ws://localhost:8080/ws');

let isRunning = false;
let isPaused = false;
let startTime = null;
let timerInterval = null;
let elapsedSeconds = 0;

const btnStart = document.getElementById('btn-start');
const btnPause = document.getElementById('btn-pause');
const btnStop = document.getElementById('btn-stop');
const startTimeEl = document.getElementById('start-time');

function updateTimer() {
    const hours = Math.floor(elapsedSeconds / 3600);
    const minutes = Math.floor((elapsedSeconds % 3600) / 60);
    const seconds = elapsedSeconds % 60;
    startTimeEl.textContent = `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
}

function updateButtons() {
    btnStart.disabled = isRunning && !isPaused;
    btnPause.disabled = !isRunning || isPaused;
    btnStop.disabled = !isRunning;
}

function startTimer() {
    if (timerInterval) clearInterval(timerInterval);
    elapsedSeconds = Math.floor((new Date() - startTime) / 1000);
    timerInterval = setInterval(() => {
        elapsedSeconds++;
        updateTimer();
    }, 1000);
}

btnStart.addEventListener('click', () => {
    ws.send(JSON.stringify({ action: 'start' }));
});

btnPause.addEventListener('click', () => {
    if (isRunning) {
        ws.send(JSON.stringify({ action: 'pause' }));
    }
});

btnStop.addEventListener('click', () => {
    ws.send(JSON.stringify({ action: 'stop' }));
});

ws.addEventListener('open', () => {
	ws.send(JSON.stringify({ action: 'update' }));
});

ws.onmessage = (event) => {
    try {
        const data = JSON.parse(event.data);
        isRunning = data.running;
        isPaused = data.paused;

        if (data.startTime) {
            startTime = new Date(data.startTime);
            if (isRunning && !isPaused) {
                startTimer();
            }
        }

        if (!isRunning || isPaused) {
            clearInterval(timerInterval);
            timerInterval = null;
        }

        updateButtons();
        updateTimer();
    } catch (e) {
        console.error("Error:", e);
    }
};

// Инициализация
startTimeEl.textContent = '00:00:00';
updateButtons();