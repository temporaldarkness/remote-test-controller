// --- WebSocket interaction (заглушка адреса) ---
const ws = new WebSocket('ws://localhost:8080/ws');

// Состояние
let isRunning = false;
let startTime = null;
let timerInterval = null;

// DOM
const btnStart = document.getElementById('btn-start');
const btnPause = document.getElementById('btn-pause');
const btnStop = document.getElementById('btn-stop');
const startTimeEl = document.getElementById('start-time');

// Формат времени HH:MM
function formatTime(date) {
    return date.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
}

// Обновить отображение времени
function updateTime() {
    if (isRunning && startTime) {
        startTimeEl.textContent = formatTime(startTime);
    } else {
        startTimeEl.textContent = '--:--';
    }
}

// Обновить кнопки
function updateButtons() {
    btnStart.disabled = isRunning;
    btnStop.disabled = !isRunning;
    btnStart.classList.toggle('active', !isRunning);
    btnStop.classList.toggle('active', isRunning);
}

// Запуск испытания
btnStart.addEventListener('click', () => {
    ws.send(JSON.stringify({ action: 'start' }));
});

// Остановка испытания
btnStop.addEventListener('click', () => {
    ws.send(JSON.stringify({ action: 'stop' }));
});

// --- WebSocket events ---
ws.onopen = () => {
    // При подключении — запросить состояние
    ws.send(JSON.stringify({ action: 'get_state' }));
};

ws.onmessage = (event) => {
    // Ожидаем сообщения вида: { running: true/false, startTime: timestamp }
    try {
        const data = JSON.parse(event.data);
        isRunning = !!data.running;
        startTime = data.startTime ? new Date(data.startTime) : null;
        updateTime();
        updateButtons();
    } catch (e) {
        // ignore
    }
};

ws.onerror = () => {
    // Ошибка соединения — отключить кнопки
    btnStart.disabled = true;
    btnStop.disabled = true;
    startTimeEl.textContent = 'нет связи';
};

ws.onclose = () => {
    btnStart.disabled = true;
    btnStop.disabled = true;
    startTimeEl.textContent = 'нет связи';
};

// --- Инициализация ---
updateTime();
updateButtons();