import { createRouter, createWebHistory } from 'vue-router';
import Dashboard from '../components/Dashboard.vue';
import AgentList from '../components/AgentList.vue';
import CommandHistory from '../components/CommandHistory.vue';
import Settings from '../components/Settings.vue';

const routes = [
    {
        path: '/',
        name: 'Dashboard',
        component: Dashboard
    },
    {
        path: '/agents',
        name: 'Agents',
        component: AgentList
    },
    {
        path: '/commands',
        name: 'Commands',
        component: CommandHistory
    },
    {
        path: '/settings',
        name: 'Settings',
        component: Settings
    }
];

const router = createRouter({
    history: createWebHistory(),
    routes
});

export default router;