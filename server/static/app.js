// DOM 요소
const agentsContainer = document.getElementById('agents');
const agentsEmpty = document.getElementById('agents-empty');
const commandInput = document.getElementById('command');
const resultsContainer = document.getElementById('results');
const resultsEmpty = document.getElementById('results-empty');

// 에이전트 데이터 저장
let agents = new Map();
let selectedAgentId = null;

// WebSocket 연결
const socket = new WebSocket(`ws://${location.host}/ws-dashboard`);

socket.onopen = () => {
    console.log('대시보드 WebSocket 연결됨');
};

socket.onmessage = (event) => {
    try {
        const msg = JSON.parse(event.data);
        handleMessage(msg);
    } catch (error) {
        console.error('메시지 파싱 오류:', error);
    }
};

socket.onclose = () => {
    console.log('대시보드 WebSocket 연결 종료');
};

socket.onerror = (error) => {
    console.error('WebSocket 오류:', error);
};

// 메시지 처리
function handleMessage(msg) {
    switch (msg.type) {
        case 'agent_list':
            handleAgentList(msg.agents);
            break;
        case 'agent_update':
            handleAgentUpdate(msg.agent);
            break;
        case 'command_result':
            handleCommandResult(msg);
            break;
        default:
            console.log('알 수 없는 메시지 타입:', msg.type);
    }
}

// 에이전트 목록 처리
function handleAgentList(agentList) {
    agents.clear();
    agentList.forEach(agent => {
        agents.set(agent.id, agent);
    });
    updateAgentsDisplay();
}

// 에이전트 업데이트 처리
function handleAgentUpdate(agent) {
    agents.set(agent.id, agent);
    updateAgentsDisplay();
}

// 에이전트 표시 업데이트
function updateAgentsDisplay() {
    if (agents.size === 0) {
        agentsContainer.style.display = 'none';
        agentsEmpty.style.display = 'block';
        return;
    }

    agentsContainer.style.display = 'block';
    agentsEmpty.style.display = 'none';
    agentsContainer.innerHTML = '';

    agents.forEach((agent, id) => {
        const card = createAgentCard(agent);
        agentsContainer.appendChild(card);
    });
}

// 에이전트 카드 생성
function createAgentCard(agent) {
    const card = document.createElement('div');
    card.className = 'agent-card';
    if (selectedAgentId === agent.id) {
        card.classList.add('selected');
    }

    const statusClass = agent.connected ? 'status-connected' : 'status-disconnected';
    const statusText = agent.connected ? '연결됨' : '연결 끊김';

    const lastSeen = new Date(agent.last_seen).toLocaleString('ko-KR');

    let statusMetrics = '';
    if (agent.status) {
        statusMetrics = `
            <div class="status-metrics">
                <div class="metric">
                    <div class="metric-label">CPU 사용률</div>
                    <div class="metric-value">${agent.status.cpu_usage.toFixed(1)}%</div>
                </div>
                <div class="metric">
                    <div class="metric-label">메모리 사용률</div>
                    <div class="metric-value">${agent.status.memory_usage.toFixed(1)} MB</div>
                </div>
                <div class="metric">
                    <div class="metric-label">디스크 사용률</div>
                    <div class="metric-value">${agent.status.disk_usage.toFixed(1)}%</div>
                </div>
                <div class="metric">
                    <div class="metric-label">업타임</div>
                    <div class="metric-value">${formatUptime(agent.status.uptime)}</div>
                </div>
            </div>
        `;
    }

    card.innerHTML = `
        <div class="agent-header">
            <div class="agent-id">${agent.info ? agent.info.hostname : agent.id}</div>
            <span class="agent-status ${statusClass}">${statusText}</span>
        </div>
        <div class="agent-info">
            <div class="agent-info-item">
                <span class="agent-info-label">ID:</span>
                <span>${agent.id}</span>
            </div>
            ${agent.info ? `
                <div class="agent-info-item">
                    <span class="agent-info-label">OS:</span>
                    <span>${agent.info.os} (${agent.info.arch})</span>
                </div>
                <div class="agent-info-item">
                    <span class="agent-info-label">MAC:</span>
                    <span>${agent.info.mac_addr}</span>
                </div>
            ` : ''}
            <div class="agent-info-item">
                <span class="agent-info-label">마지막 확인:</span>
                <span>${lastSeen}</span>
            </div>
        </div>
        ${statusMetrics}
    `;

    // 카드 클릭 시 선택
    card.addEventListener('click', () => {
        if (agent.connected) {
            // 기존 선택 해제
            document.querySelectorAll('.agent-card').forEach(c => {
                c.classList.remove('selected');
            });
            
            if (selectedAgentId === agent.id) {
                selectedAgentId = null;
            } else {
                selectedAgentId = agent.id;
                card.classList.add('selected');
            }
        }
    });

    return card;
}

// 업타임 포맷팅
function formatUptime(seconds) {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    
    if (days > 0) {
        return `${days}일 ${hours}시간`;
    } else if (hours > 0) {
        return `${hours}시간 ${minutes}분`;
    } else {
        return `${minutes}분`;
    }
}

// 명령 전송
function sendCommand() {
    const command = commandInput.value.trim();
    if (!command) {
        alert('명령어를 입력하세요.');
        return;
    }

    const target = document.querySelector('input[name="target"]:checked').value;
    
    const msg = {
        type: 'command',
        command: command
    };

    if (target === 'selected' && selectedAgentId) {
        msg.agent_id = selectedAgentId;
    } else if (target === 'selected' && !selectedAgentId) {
        alert('에이전트를 선택하세요.');
        return;
    }

    socket.send(JSON.stringify(msg));
    commandInput.value = '';
}

// Enter 키로 명령 전송
commandInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') {
        sendCommand();
    }
});

// 명령 실행 결과 처리
function handleCommandResult(msg) {
    if (resultsEmpty) {
        resultsEmpty.style.display = 'none';
    }
    if (resultsContainer) {
        resultsContainer.style.display = 'block';
    }

    const resultItem = document.createElement('div');
    resultItem.className = 'result-item';

    const timestamp = new Date(msg.result.timestamp).toLocaleString('ko-KR');
    const agentName = agents.get(msg.agent_id)?.info?.hostname || msg.agent_id;

    let errorSection = '';
    if (msg.result.error) {
        errorSection = `<div class="result-error">오류: ${msg.result.error}</div>`;
    }

    resultItem.innerHTML = `
        <div class="result-header">
            <div>
                <span class="result-agent">${agentName}</span>
                <span class="result-command">${msg.result.command}</span>
            </div>
            <div class="result-timestamp">${timestamp}</div>
        </div>
        <div class="result-output">${escapeHtml(msg.result.output)}</div>
        ${errorSection}
        <div style="margin-top: 10px; color: #666; font-size: 0.9em;">
            종료 코드: ${msg.result.exit_code}
        </div>
    `;

    // 최신 결과를 맨 위에 추가
    if (resultsContainer.firstChild) {
        resultsContainer.insertBefore(resultItem, resultsContainer.firstChild);
    } else {
        resultsContainer.appendChild(resultItem);
    }

    // 결과가 너무 많으면 오래된 것 제거 (최대 50개)
    while (resultsContainer.children.length > 50) {
        resultsContainer.removeChild(resultsContainer.lastChild);
    }
}

// HTML 이스케이프
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
