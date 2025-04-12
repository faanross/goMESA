<template>
  <div class="execute-tab">
    <h1>Reflective Loading</h1>

    <div class="card">
      <div class="card-header">
        <h2>Load DLL in Memory</h2>
      </div>

      <div class="form-group">
        <label for="agent-select">Target Agent</label>
        <select
            id="agent-select"
            v-model="selectedAgentId"
            :disabled="isExecuting"
        >
          <option value="">Select an agent</option>
          <option
              v-for="agent in aliveAgents"
              :key="agent.ID"
              :value="agent.ID"
          >
            {{ agent.ID }} ({{ agent.OS || 'Unknown OS' }})
          </option>
        </select>
      </div>

      <div class="form-group">
        <label for="dll-file">DLL Payload</label>
        <div class="file-upload">
          <input
              type="file"
              id="dll-file"
              ref="fileInput"
              @change="handleFileChange"
              :disabled="isExecuting"
              accept=".dll"
          />
          <div class="file-info" v-if="selectedFile">
            {{ selectedFile.name }} ({{ formatFileSize(selectedFile.size) }})
          </div>
        </div>
      </div>

      <div class="form-group">
        <label for="function-name">Function Name to Execute</label>
        <input
            type="text"
            id="function-name"
            v-model="functionName"
            placeholder="Enter exported function name (e.g., LaunchCalc)"
            :disabled="isExecuting"
        />
      </div>

      <div class="form-actions">
        <button
            class="primary"
            @click="executeReflectiveLoading"
            :disabled="!canExecute || isExecuting"
        >
          {{ isExecuting ? 'Executing...' : 'Execute' }}
        </button>
      </div>
    </div>

    <!-- Results Card -->
    <div class="card mt-3" v-if="executionResult">
      <div class="card-header">
        <h2>Execution Result</h2>
      </div>

      <div class="execution-result" :class="resultClass">
        <div class="result-status">
          {{ executionResult.success ? 'SUCCESS' : 'FAILED' }}
        </div>
        <div class="result-message">
          {{ executionResult.message }}
        </div>
        <div class="result-time">
          {{ formatTime(executionResult.timestamp) }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { agents, sendMessage, addMessageListener } from '../services/websocket';

// State
const selectedAgentId = ref('');
const selectedFile = ref(null);
const functionName = ref('');
const isExecuting = ref(false);
const executionResult = ref(null);
const fileInput = ref(null);

// Computed properties
const aliveAgents = computed(() => {
  return agents.filter(agent => agent.Status === 'ALIVE');
});

const canExecute = computed(() => {
  return selectedAgentId.value && selectedFile.value && functionName.value;
});

const resultClass = computed(() => {
  if (!executionResult.value) return '';
  return executionResult.value.success ? 'success' : 'error';
});

// Methods
function handleFileChange(event) {
  const files = event.target.files;
  if (files.length > 0) {
    selectedFile.value = files[0];
  } else {
    selectedFile.value = null;
  }
}

function formatFileSize(bytes) {
  if (bytes < 1024) {
    return bytes + ' bytes';
  } else if (bytes < 1024 * 1024) {
    return (bytes / 1024).toFixed(1) + ' KB';
  } else {
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  }
}

function formatTime(timestamp) {
  if (!timestamp) return '';
  const date = new Date(timestamp);
  return date.toLocaleString();
}

async function executeReflectiveLoading() {
  if (!canExecute.value || isExecuting.value) return;

  isExecuting.value = true;
  executionResult.value = null;

  try {
    // Read the file as an ArrayBuffer
    const fileReader = new FileReader();

    // Create a promise to handle the FileReader
    const fileLoaded = new Promise((resolve, reject) => {
      fileReader.onload = () => resolve(fileReader.result);
      fileReader.onerror = () => reject(fileReader.error);
    });

    fileReader.readAsArrayBuffer(selectedFile.value);

    // Wait for the file to be read
    const arrayBuffer = await fileLoaded;

    // Convert to base64 for transmission
    const base64Data = btoa(
        new Uint8Array(arrayBuffer)
            .reduce((data, byte) => data + String.fromCharCode(byte), '')
    );

    // Send to server
    sendMessage({
      type: 'executeReflectiveLoading',
      agentId: selectedAgentId.value,
      payload: base64Data,
      functionName: functionName.value
    });

    // Message will be processed by our listener and executionResult will be updated
  } catch (error) {
    console.error('Error preparing file for upload:', error);

    executionResult.value = {
      success: false,
      message: `Error preparing file: ${error.message}`,
      timestamp: new Date()
    };

    isExecuting.value = false;
  }
}

// Lifecycle
onMounted(() => {
  // Set up listener for reflective loading responses
  const removeListener = addMessageListener('reflectiveLoadingResponse', (message) => {
    isExecuting.value = false;

    executionResult.value = {
      success: message.success,
      message: message.message || (message.success ? 'Operation completed successfully' : 'Operation failed'),
      timestamp: message.timestamp || new Date()
    };

    // Reset file input to allow selecting the same file again
    if (fileInput.value) {
      fileInput.value.value = '';
    }
  });

  // Clean up on component unmount
  return () => {
    removeListener();
  };
});
</script>

<style scoped>
.execute-tab h1 {
  color: var(--purple);
  margin-bottom: 2rem;
}

.file-upload {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.file-info {
  font-size: 0.9rem;
  color: var(--comment);
  padding: 0.5rem;
  background-color: var(--background);
  border-radius: 4px;
}

.form-actions {
  margin-top: 2rem;
  display: flex;
  justify-content: flex-end;
}

.execution-result {
  padding: 1.5rem;
  border-radius: 4px;
  background-color: var(--background);
}

.execution-result.success {
  border-left: 4px solid var(--green);
}

.execution-result.error {
  border-left: 4px solid var(--red);
}

.result-status {
  font-weight: bold;
  font-size: 1.2rem;
  margin-bottom: 0.5rem;
}

.success .result-status {
  color: var(--green);
}

.error .result-status {
  color: var(--red);
}

.result-message {
  margin-bottom: 1rem;
  white-space: pre-wrap;
  font-family: 'Courier New', monospace;
}

.result-time {
  font-size: 0.85rem;
  color: var(--comment);
  text-align: right;
}
</style>