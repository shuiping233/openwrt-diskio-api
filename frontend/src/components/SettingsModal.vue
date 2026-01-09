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
  <Transition
    enter-active-class="transition-opacity duration-200 ease-out"
    enter-from-class="opacity-0"
    enter-to-class="opacity-100"
    leave-active-class="transition-opacity duration-200 ease-in"
    leave-from-class="opacity-100"
    leave-to-class="opacity-0"
  >
    <div v-if="isOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-none">
      <!-- 模态框主体 -->
      <div 
        class="bg-slate-800 rounded-xl border border-slate-700 w-full max-w-lg shadow-2xl relative transform transition-all"
        @click.stop
      >
        
        <!-- 头部: 标题 + 关闭按钮 (右上角 X) -->
        <div class="flex justify-between items-center px-6 py-4 border-b border-slate-700">
          <h2 class="text-xl font-bold text-white">系统设置</h2>
          <button 
            @click="emit('update:isOpen', false)"
            class="text-slate-400 hover:text-white transition-colors p-1 rounded hover:bg-slate-700"
            title="关闭 (Esc)"
          >
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
                <input 
                  type="number" 
                  min="1" 
                  max="365"
                  v-model.number="retentionDays" 
                  @change="handleSave"
                  class="bg-slate-900 border border-slate-600 rounded px-3 py-1.5 w-24 text-white outline-none focus:border-blue-500 transition-colors"
                />
              </div>
              <p class="text-xs text-slate-500">
                超过此天数的历史图表数据将被自动清理。
              </p>

              <!-- 配置 2: 清空所有数据 -->
              <div>
                <button 
                  @click="handleClear"
                  class="w-full mt-2 bg-red-600 hover:bg-red-500 text-white py-2 rounded transition-colors text-sm font-semibold flex items-center justify-center gap-2"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h4a1 1 0 001 1v3M4 7h16" />
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
          <button 
            @click="emit('update:isOpen', false)"
            class="px-6 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded transition-colors text-sm font-medium"
          >
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