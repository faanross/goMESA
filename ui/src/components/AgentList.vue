<template>
  <div class="agent-list">
    <div class="header-actions">
      <h1>Agents</h1>
      <div class="actions">
        <button class="info" @click="refreshAgents">
          Refresh
        </button>
      </div>
    </div>

    <div class="filters mb-3">
      <div class="form-group">
        <input
            v-model="searchTerm"
            placeholder="Search agents..."
            class="search-input"
        />
      </div>

      <div class="filter-buttons">
        <button
            @click="activeFilter = 'all'"
            :class="{ active: activeFilter === 'all' }"
        >
          All
        </button>
        <button
            @click="activeFilter = 'alive'"
            :class="{ active: activeFilter === 'alive' }"
        >
          Alive
        </button>
        <button
            @click="activeFilter = 'mia'"
            :class="{ active: activeFilter === 'mia' }"
        >
          MIA
        </button>
        <button
            @click="activeFilter = 'killed'"
            :class="{ active: activeFilter === 'killed' }"
        >
          Killed
        </button>
      </div>
    </div>

    <div class="table-responsive">
      <table v-if="filteredAgents.length > 0">
        <thead>
        <tr>
          <th>Agent ID</th>
          <th>OS</th>
          <th>Service/Group</th>
          <th>Status</th>
          <th>Last Seen</th>
          <th>Actions</th>
        </tr>
        </thead>
        <tbody>
        <tr v-for="agent in filteredAgents" :key="agent.ID">
          <td>{{ agent.ID }}</td>
          <td>{{ agent.OS || 'Unknown' }}</td>
          <td>{{ agent.Service || 'None' }}</td>
          <td>
              <span class="status-badge" :class="getStatusClass(agent.Status)">
                {{ agent.Status }}
              </span>
          </td>
          <td>{{ formatTime(agent.LastSeen) }}</td>
          <td>
            <div class="agent-actions">
              <button
                  class="action-btn info"
                  title="Ping Agent"
                  @click="pingAgent(agent.ID)"
                  :disabled="agent.Status !== 'ALIVE'"
              >
                📡
              </button>
              <button
                  class="action-btn primary"
                  title="Execute Command"
                  @click="openCommandModal(agent)"
                  :disabled="agent.Status !== 'ALIVE'"
              >
                💻
              </button>
              <button
                  class="action-btn warning"
                  title="Group Agent"
                  @click="openGroupModal(agent)"
              >
                🏷️
              </button>
              <button
                  class="action-btn danger"
                  title="Kill Agent"
                  @click="openKillModal(agent)"
                  :disabled="agent.Status !== 'ALIVE'"
              >
                ⚠️
              </button>
            </div>
          </td>
        </tr>
        </tbody>
      </table>
      <div v-else class="empty-state">
        No agents match your filters.
      </div>
    </div>

    <!-- Command Modal -->
    <div v-if="showCommandModal" class="modal-overlay" @click="showCommandModal = false">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h2>Execute Command on {{ selectedAgent?.ID }}</h2>
          <button class="modal-close" @click="showCommandModal = false">×</button>
        </div>

        <div class="form-group">
          <label for="command-input">Command</label>
          <input
              id="command-input"
              v-model="commandInput"
              placeholder="Enter command..."
              @keyup.enter="executeCommand"
          />
        </div>

        <div v-if="commandOutput" class="form-group">
          <label>Output</label>
          <div class="console">
            <pre>{{ commandOutput }}</pre>
          </div>
        </div>

        <div class="modal-footer">
          <button class="danger" @click="showCommandModal = false">Close</button>
          <button
              class="primary"
              @click="executeCommand"
              :disabled="!commandInput || commandLoading"
          >
            {{ commandLoading ? 'Executing...' : 'Execute' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Group Modal -->
    <div v-if="showGroupModal" class="modal-overlay" @click="showGroupModal = false">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h2>Group Agent {{ selectedAgent?.ID }}</h2>
          <button class="modal-close" @click="showGroupModal = false">×</button>
        </div>

        <div class="form-group">
          <label for="group-name">Group Name</label>
          <input
              id="group-name"
              v-model="groupInput"
              placeholder="Enter group name..."
              @keyup.enter="setAgentGroup"
          />
        </div>

        <div class="modal-footer">
          <button class="danger" @click="showGroupModal = false">Cancel</button>
          <button
              class="primary"
              @click="setAgentGroup"
              :disabled="!groupInput"
          >
            Save
          </button>
        </div>
      </div>
    </div>

    <!-- Kill Confirmation Modal -->
    <div v-if="showKillModal" class="modal-overlay" @click="showKillModal = false">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h2>Kill Agent</h2>
          <button class="modal-close" @click="showKillModal = false">×</button>
        </div>

        <div class="kill-warning">
          <p>Are you sure you want to kill agent <strong>{{ selectedAgent?.ID }}</strong>?</p>
          <p>This will terminate the agent process on the target system.</p>
        </div>

        <div class="modal-footer">
          <button class="info" @click="showKillModal = false">Cancel</button>
          <button class="danger" @click="killSelectedAgent">
            Confirm Kill
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import {
  agents,
  sendMessage,
  addMessageListener,
  pingAgent as wsPingAgent,
  killAgent as wsKillAgent,
  groupAgent as wsGroupAgent,
  executeCommand as wsExecuteCommand
} from '../services/websocket';

// State
const searchTerm = ref('');
const activeFilter = ref('all');
const showCommandModal = ref(false);
const showGroupModal = ref(false);
const showKillModal = ref(false);
const selectedAgent = ref(null);
const commandInput = ref('');
const commandOutput = ref('');
const commandLoading = ref(false);
const groupInput = ref('');

// Computed
const filteredAgents = computed(() => {
  let filtered = [...agents];

  // Apply search filter
  if (searchTerm.value) {
    const term = searchTerm.value.toLowerCase();
    filtered = filtered.filter(agent =>
        agent.ID.toLowerCase().includes(term) ||
        (agent.OS && agent.OS.toLowerCase().includes(term)) ||
        (agent.Service && agent.Service.toLowerCase().includes(term))
    );
  }

  // Apply status filter
  if (activeFilter.value !== 'all') {
    const statusMap = {
      'alive': 'ALIVE',
      'mia': 'MIA',
      'killed': 'SRV-KILLED'
    };

    filtered = filtered.filter(agent =>
        agent.Status === statusMap[activeFilter.value]
    );
  }

  return filtered;
});

// Methods
function refreshAgents() {
  sendMessage({ type: 'getAgents' });
}

function formatTime(timestamp) {
  if (!timestamp) return 'Unknown';

  // Convert to local date string
  const date = new Date(timestamp);
  return date.toLocaleString();
}

function getStatusClass(status) {
  switch (status) {
    case 'ALIVE': return 'status-alive';
    case 'MIA': return 'status-mia';
    case 'SRV-KILLED': return 'status-killed';
    default: return '';
  }
}

function pingAgent(agentId) {
  wsPingAgent(agentId);
}

function openCommandModal(agent) {
  selectedAgent.value = agent;
  commandInput.value = '';
  commandOutput.value = '';
  showCommandModal.value = true;
}

function openGroupModal(agent) {
  selectedAgent.value = agent;
  groupInput.value = agent.Service || '';
  showGroupModal.value = true;
}

function openKillModal(agent) {
  selectedAgent.value = agent;
  showKillModal.value = true;
}

function executeCommand() {
  if (!commandInput.value || !selectedAgent.value) return;

  commandLoading.value = true;
  commandOutput.value = 'Executing command...';

  wsExecuteCommand(selectedAgent.value.ID, commandInput.value);
}

function setAgentGroup() {
  if (!groupInput.value || !selectedAgent.value) return;

  wsGroupAgent(selectedAgent.value.ID, groupInput.value);
  showGroupModal.value = false;
}

function killSelectedAgent() {
  if (!selectedAgent.value) return;

  wsKillAgent(selectedAgent.value.ID);
  showKillModal.value = false;
}

// Lifecycle hooks
onMounted(() => {
  // Set up listener for command responses
  const removeListener = addMessageListener('commandResponse', (message) => {
    if (showCommandModal.value && selectedAgent.value &&
        message.agentId === selectedAgent.value.ID) {
      commandLoading.value = false;

      if (message.success) {
        commandOutput.value = message.output || 'Command executed successfully (no output)';
      } else {
        commandOutput.value = `Error: ${message.error || 'Unknown error'}`;
      }
    }
  });

  // Refresh agents list
  refreshAgents();

  // Clean up listener on component unmount
  return () => {
    removeListener();
  };
});
</script>

<style scoped>
.agent-list {
  position: relative;
}

.header-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
}

.header-actions h1 {
  color: var(--purple);
  margin: 0;
}

.filters {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  align-items: center;
}

.search-input {
  padding: 0.75rem 1rem;
  border-radius: 4px;
  border: 1px solid var(--comment);
  background-color: var(--background);
  color: var(--foreground);
  min-width: 300px;
}

.filter-buttons {
  display: flex;
  gap: 0.5rem;
}

.filter-buttons button {
  padding: 0.5rem 1rem;
  background-color: var(--background);
  color: var(--foreground);
  border: 1px solid var(--comment);
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.3s;
}

.filter-buttons button:hover {
  border-color: var(--purple);
}

.filter-buttons button.active {
  background-color: var(--purple);
  color: var(--background);
  border-color: var(--purple);
}

.table-responsive {
  overflow-x: auto;
}

.empty-state {
  padding: 3rem;
  text-align: center;
  background-color: var(--selection);
  border-radius: 8px;
  margin-top: 2rem;
  color: var(--comment);
}

.status-badge {
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.85rem;
  font-weight: bold;
}

.status-alive {
  background-color: var(--green);
  color: var(--background);
}

.status-mia {
  background-color: var(--orange);
  color: var(--background);
}

.status-killed {
  background-color: var(--red);
  color: var(--foreground);
}

.agent-actions {
  display: flex;
  gap: 0.5rem;
}

.action-btn {
  background: none;
  border: none;
  font-size: 1.25rem;
  cursor: pointer;
  padding: 0.25rem;
  border-radius: 4px;
  transition: transform 0.2s;
}

.action-btn:hover:not(:disabled) {
  transform: scale(1.2);
}

.action-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.action-btn.info {
  color: var(--cyan);
}

.action-btn.primary {
  color: var(--purple);
}

.action-btn.warning {
  color: var(--orange);
}

.action-btn.danger {
  color: var(--red);
}

.kill-warning {
  padding: 1.5rem;
  color: var(--red);
  text-align: center;
}

.kill-warning p {
  margin-bottom: 1rem;
}

.kill-warning strong {
  font-weight: bold;
}
</style>