<template>
  <div class="app-container">
    <header>
      <h1>goMESA C2 Framework</h1>
      <div class="connection-status" :class="connectionState">
        {{ connectionState }}
        <button v-if="connectionState === 'disconnected'" class="reconnect-btn" @click="reconnect">
          Reconnect
        </button>
      </div>
    </header>

    <nav>
      <router-link to="/">Dashboard</router-link>
      <router-link to="/agents">Agents</router-link>
      <router-link to="/commands">Commands</router-link>
      <router-link to="/settings">Settings</router-link>
    </nav>

    <main>
      <router-view></router-view>
    </main>

    <notifications position="top right" />
  </div>
</template>

<script setup>
import { onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { connectWebSocket, connectionState } from './services/websocket';

const router = useRouter();

// Function to reconnect manually
function reconnect() {
  connectWebSocket();
}

onMounted(() => {
  // Connect to WebSocket only once when the application first loads
  connectWebSocket();
});
</script>
<style>
:root {
  /* Dracula theme colors */
  --background: #282a36;
  --foreground: #f8f8f2;
  --selection: #44475a;
  --comment: #6272a4;
  --purple: #bd93f9;
  --green: #50fa7b;
  --orange: #ffb86c;
  --pink: #ff79c6;
  --red: #ff5555;
  --yellow: #f1fa8c;
  --cyan: #8be9fd;
}

* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

body {
  font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
  background-color: var(--background);
  color: var(--foreground);
  line-height: 1.6;
}

.app-container {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 2rem;
  background-color: var(--selection);
  border-bottom: 2px solid var(--purple);
}

header h1 {
  color: var(--purple);
  font-weight: bold;
}

.connection-status {
  padding: 0.5rem 1rem;
  border-radius: 4px;
  font-weight: bold;
}

.connection-status.connected {
  background-color: var(--green);
  color: var(--background);
}

.connection-status.connecting {
  background-color: var(--yellow);
  color: var(--background);
}

.connection-status.disconnected {
  background-color: var(--red);
  color: var(--foreground);
}

nav {
  display: flex;
  padding: 0.5rem 2rem;
  background-color: var(--selection);
  border-bottom: 1px solid var(--comment);
}

nav a {
  padding: 0.5rem 1rem;
  margin-right: 1rem;
  color: var(--foreground);
  text-decoration: none;
  border-radius: 4px;
  transition: background-color 0.3s, color 0.3s;
}

nav a:hover, nav a.router-link-active {
  background-color: var(--purple);
  color: var(--background);
}

main {
  flex: 1;
  padding: 2rem;
  overflow-y: auto;
}

/* Button styles */
button {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 4px;
  font-weight: bold;
  cursor: pointer;
  transition: background-color 0.3s, transform 0.1s;
}

button:hover {
  transform: translateY(-2px);
}

button:active {
  transform: translateY(0);
}

button.primary {
  background-color: var(--purple);
  color: var(--background);
}

button.success {
  background-color: var(--green);
  color: var(--background);
}

button.warning {
  background-color: var(--orange);
  color: var(--background);
}

button.danger {
  background-color: var(--red);
  color: var(--foreground);
}

button.info {
  background-color: var(--cyan);
  color: var(--background);
}

/* Card styles */
.card {
  background-color: var(--selection);
  border-radius: 8px;
  padding: 1.5rem;
  margin-bottom: 1.5rem;
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
}

.card-header {
  margin-bottom: 1rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header h2 {
  color: var(--pink);
  font-size: 1.5rem;
}

/* Form styles */
.form-group {
  margin-bottom: 1rem;
}

label {
  display: block;
  margin-bottom: 0.5rem;
  color: var(--cyan);
}

input, textarea, select {
  width: 100%;
  padding: 0.75rem;
  border-radius: 4px;
  border: 1px solid var(--comment);
  background-color: var(--background);
  color: var(--foreground);
  font-size: 1rem;
}

input:focus, textarea:focus, select:focus {
  outline: none;
  border-color: var(--purple);
  box-shadow: 0 0 0 2px rgba(189, 147, 249, 0.3);
}

/* Table styles */
table {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 1.5rem;
}

thead {
  background-color: var(--selection);
}

th {
  text-align: left;
  padding: 1rem;
  color: var(--pink);
  font-weight: bold;
}

td {
  padding: 1rem;
  border-bottom: 1px solid var(--comment);
}

tr:hover {
  background-color: rgba(68, 71, 90, 0.5);
}

/* Grid and flex helpers */
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1.5rem;
}

.flex {
  display: flex;
  gap: 1rem;
}

.flex-between {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.flex-column {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

/* Spacing helpers */
.mt-1 { margin-top: 0.5rem; }
.mt-2 { margin-top: 1rem; }
.mt-3 { margin-top: 1.5rem; }
.mt-4 { margin-top: 2rem; }

.mb-1 { margin-bottom: 0.5rem; }
.mb-2 { margin-bottom: 1rem; }
.mb-3 { margin-bottom: 1.5rem; }
.mb-4 { margin-bottom: 2rem; }

/* Console styles */
.console {
  background-color: var(--background);
  border: 1px solid var(--comment);
  border-radius: 4px;
  padding: 1rem;
  font-family: 'Courier New', Courier, monospace;
  overflow-x: auto;
  white-space: pre-wrap;
  line-height: 1.4;
}

.console pre {
  margin: 0;
}
</style>