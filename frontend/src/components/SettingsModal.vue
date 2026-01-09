<script setup lang="ts">
import { ref, onMounted, watch } from 'vue';
import { useDatabase } from '../useDatabase';
import { useToast } from '../useToast';

// 定义 Props (支持 v-model)
const props = defineProps<{
  isOpen: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:isOpen', value: boolean): void;
}>();

// 逻辑
const { getConfig, setConfig, clearHistory } = useDatabase();
const { success, error } = useToast();

const retentionDays = ref(7);

// 加载配置
onMounted(async () => {
  const days = await getConfig<number>('retention_days');
  if (days) retentionDays.value = days;
});

// 保存配置 (即时生效)
const handleSave = async () => {
  await setConfig('retention_days', retentionDays.value);
  success('保存天数设置已更新');
};

// 清空数据
const handleClear = async () => {
  if (confirm('警告：确定清空所有历史图表数据吗？此操作不可恢复。')) {
    await clearHistory();
    success('历史数据已清空');
  }
};

// ESC 键关闭
const handleKeydown = (e: KeyboardEvent) => {
  if (props.isOpen && e.key === 'Escape') {
    emit('update:isOpen', false);
  }
};

// 监听打开状态，注册/注销全局键盘事件
watch(() => props.isOpen, (newVal) => {
  if (newVal) {
    window.addEventListener('keydown', handleKeydown);
  } else {
    window.removeEventListener('keydown', handleKeydown);
  }
});
</script>

<template>
  <!-- 遮罩层 (无高斯模糊，半透明黑色背景) -->
  <Transition enter-active-class="transition-opacity duration-200 ease-out" enter-from-class="opacity-0"
    enter-to-class="opacity-100" leave-active-class="transition-opacity duration-200 ease-in"
    leave-from-class="opacity-100" leave-to-class="opacity-0">
    <div v-if="isOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-none">
      <!-- 模态框主体 -->
      <div
        class="bg-slate-800 rounded-xl border border-slate-700 w-full max-w-lg shadow-2xl relative transform transition-all"
        @click.stop>

        <!-- 头部: 标题 + 关闭按钮 (右上角 X) -->
        <div class="flex justify-between items-center px-6 py-4 border-b border-slate-700">
          <h2 class="text-xl font-bold text-white">系统设置</h2>
          <button @click="emit('update:isOpen', false)"
            class="text-slate-400 hover:text-white transition-colors p-1 rounded hover:bg-slate-700" title="关闭 (Esc)">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
          </button>
        </div>

        <!-- 内容区域 -->
        <div class="px-6 py-6 space-y-6">

          <!-- 分类 1: 历史数据监控 -->
          <div>
            <h3 class="text-lg font-bold text-blue-400 mb-1 pb-2 border-b border-slate-700">
              历史数据监控
            </h3>

            <div class="mt-4 space-y-4">
              <!-- 配置 1: 数据保存天数 -->
              <div class="flex justify-between items-center">
                <label class="text-slate-300 text-sm">数据保留天数</label>
                <input type="number" min="1" max="365" v-model.number="retentionDays" @change="handleSave"
                  class="bg-slate-900 border border-slate-600 rounded px-3 py-1.5 w-24 text-white outline-none focus:border-blue-500 transition-colors" />
              </div>
              <p class="text-xs text-slate-500">
                超过此天数的历史图表数据将被自动清理。
              </p>

              <!-- 配置 2: 清空所有数据 -->
              <div>
                <button @click="handleClear"
                  class="w-full mt-2 bg-red-600 hover:bg-red-500 text-white py-2 rounded transition-colors text-sm font-semibold flex items-center justify-center gap-2">
                  <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" fill="currentColor" class="bi bi-trash"
                    viewBox="0 0 16 16">
                    <path
                      d="M5.5 5.5A.5.5 0 0 1 6 6v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5m2.5 0a.5.5 0 0 1 .5.5v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5m3 .5a.5.5 0 0 0-1 0v6a.5.5 0 0 0 1 0z" />
                    <path
                      d="M14.5 3a1 1 0 0 1-1 1H13v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V4h-.5a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1H6a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1h3.5a1 1 0 0 1 1 1zM4.118 4 4 4.059V13a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1V4.059L11.882 4zM2.5 3h11V2h-11z" />
                  </svg>
                  清空所有数据
                </button>
              </div>
            </div>
          </div>

          <!-- 预留：可在此添加更多分类 -->
          <!-- <div>
             <h3 class="text-lg font-bold text-green-400 mb-1 pb-2 border-b border-slate-700">
               其他设置
             </h3>
          </div> -->

        </div>

        <!-- 底部: 退出按钮 (右下角) -->
        <div class="px-6 py-4 border-t border-slate-700 flex justify-end bg-slate-800/50 rounded-b-xl">
          <button @click="emit('update:isOpen', false)"
            class="px-6 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded transition-colors text-sm font-medium">
            退出
          </button>
        </div>

      </div>
    </div>
  </Transition>
</template>

<style scoped>
/* 确保过渡动画流畅 */
</style>