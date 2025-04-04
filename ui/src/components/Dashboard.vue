<template>
  <div class="dashboard">
    <h1>Dashboard</h1>

    <div class="grid">
      <div class="card">
        <div class="card-header">
          <h2>Agent Overview</h2>
        </div>
        <div class="stats">
          <div class="stat-item">
            <div class="stat-value">{{ agentStats.total }}</div>
            <div class="stat-label">Total Agents</div>
          </div>
          <div class="stat-item">
            <div class="stat-value">{{ agentStats.alive }}</div>
            <div class="stat-label">Active Agents</div>
          </div>
          <div class="stat-item">
            <div class="stat-value">{{ agentStats.mia }}</div>
            <div class="stat-label">MIA Agents</div>
          </div>
          <div class="stat-item">
            <div class="stat-value">{{ agentStats.killed }}</div>
            <div class="stat-label">Killed Agents</div>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="card-header">
          <h2>Recent Commands</h2>
        </div>
        <div v-if="commandHistory.length > 0">
          <div v-for="(cmd, index) in recentCommands" :key="index" class="command-item">
            <div class="command-content">
              <div class="command-text">{{ cmd.command }}</div>
              <div class="command-agent">Agent: {{ cmd.agentId }}</div>
            </div>
            <div class="command-status" :class="cmd.success ? 'success' : 'error'">
              {{ cmd.success ? 'Success' : 'Failed' }}
            </div>
          </div>
        </div>
        <div v-else class="empty-state">
          No command history available.
        </div>
      </div>
    </div>

    <div class="card mt-4">
      <div class="card-header">
        <h2>Quick Actions</h2>
      </div>
      <div class="quick-actions">
        <router-link to="/agents" class="quick-action">
          <div class="action-icon">👥</div>
          <div class="action-text">View All Agents</div>
        </router-link>

        <router-link to="/commands" class="quick-action">
          <div class="action-icon">💻</div>
          <div class="action-text">Command History</div>
        </router-link>

        <div class="quick-action" @click="showQuickExecute = true">
          <div class="action-icon">📡</div>
          <div class="action-text">Quick Execute</div>
        </div>
      </div>
    </div>

    <!-- Quick Execute Modal -->
    <div v-if="showQuickExecute" class="modal-overlay" @click="showQuickExecute = false">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h2>Quick Execute Command</h2>
          <button class="modal-close" @click="showQuickExecute = false">×</button>
        </div>

        <div class="form-group">
          <label for="quick-agent">Select Agent</label>
          <select id="quick-agent" v-model="quickExecute.agentId">
            <option disabled value="">Select an agent</option>
            <option v-for="agent in aliveAgents" :key="agent.ID" :value="agent.ID">
              {{ agent.ID }} ({{ agent.OS }})
            </option>
          </select>
        </div>

        <div class="form-group">
          <label for="quick-command">Command</label>
          <input id="quick-command" v-model="quickExecute.command" placeholder="Enter command..." />
        </div>

        <div class="modal-footer">
          <button class="danger" @click="showQuickExecute = false">Cancel</button>
          <button class="primary" @click="executeQuickCommand" :disabled="!quickExecute.agentId || !quickExecute.command">
            Execute
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { agents, commandHistory, executeCommand } from '../services/websocket';

// Quick execute modal state
const showQuickExecute = ref(false);
const quickExecute = ref({
  agentId: '',
  command: '',
});

// Computed properties
const agentStats = computed(() => {
  return {
    total: agents.length,
    alive: agents.filter(a => a.Status === 'ALIVE').length,
    mia: agents.filter(a => a.Status === 'MIA').length,
    killed: agents.filter(a => a.Status === 'SRV-KILLED').length,
  };
});

const recentCommands = computed(() => {
  return commandHistory.slice(0, 5);
});

const aliveAgents = computed(() => {
  return agents.filter(a => a.Status === 'ALIVE');
});

// Execute quick command
function executeQuickCommand() {
  if (quickExecute.value.agentId && quickExecute.value.command) {
    executeCommand(quickExecute.value.agentId, quickExecute.value.command);

    // Reset form and close modal
    quickExecute.value.command = '';
    showQuickExecute.value = false;
  }
}
</script>

<style scoped>
.dashboard h1 {
  margin-bottom: 2rem;
  color: var(--purple);
}

.stats {
  display: flex;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 1rem;
}

.stat-item {
  flex: 1;
  min-width: 100px;
  background-color: var(--background);
  padding: 1rem;
  border-radius: 8px;
  text-align: center;
}

.stat-value {
  font-size: 2rem;
  font-weight: bold;
  color: var(--pink);
  margin-bottom: 0.5rem;
}

.stat-label {
  color: var(--comment);
}

.command-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  background-color: var(--background);
  border-radius: 4px;
  margin-bottom: 0.5rem;
}

.command-text {
  font-family: 'Courier New', Courier, monospace;
  color: var(--yellow);
  margin-bottom: 0.25rem;
}

.command-agent {
  font-size: 0.85rem;
  color: var(--comment);
}

.command-status {
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.85rem;
  font-weight: bold;
}

.command-status.success {
  background-color: var(--green);
  color: var(--background);
}

.command-status.error {
  background-color: var(--red);
  color: var(--foreground);
}

.empty-state {
  padding: 2rem;
  text-align: center;
  color: var(--comment);
  background-color: var(--background);
  border-radius: 4px;
}

.quick-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
}

.quick-action {
  flex: 1;
  min-width: 150px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background-color: var(--background);
  padding: 1.5rem;
  border-radius: 8px;
  text-decoration: none;
  color: var(--foreground);
  cursor: pointer;
  transition: transform 0.3s, box-shadow 0.3s;
}

.quick-action:hover {
  transform: translateY(-4px);
  box-shadow: 0 6px 12px rgba(0, 0, 0,.2);
}

.action-icon {
  font-size: 2rem;
  margin-bottom: 0.5rem;
}

.action-text {
  text-align: center;
  font-weight: bold;
}

/* Modal styles */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.7);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal-content {
  background-color: var(--selection);
  border-radius: 8px;
  width: 90%;
  max-width: 500px;
  max-height: 90vh;
  overflow-y: auto;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid var(--comment);
}

.modal-header h2 {
  color: var(--pink);
  margin: 0;
}

.modal-close {
  background: none;
  border: none;
  color: var(--foreground);
  font-size: 1.5rem;
  cursor: pointer;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 1rem;
  padding: 1.5rem;
  border-top: 1px solid var(--comment);
}

.form-group {
  padding: 0 1.5rem;
  margin-top: 1.5rem;
}
</style>