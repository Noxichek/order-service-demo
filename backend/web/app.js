document.getElementById('fetchBtn').addEventListener('click', async () => {
    const orderId = document.getElementById('orderId').value.trim();
    const errorDiv = document.getElementById('error');
    const resultPre = document.getElementById('result');
    errorDiv.textContent = '';
    resultPre.textContent = '';
    if (!orderId) return;

    try {
        const response = await fetch(`/order/${orderId}`);
        if (!response.ok) {
            if (response.status === 404) {
                errorDiv.textContent = 'Заказ не найден';
            } else {
                errorDiv.textContent = 'Ошибка сервера';
            }
            return;
        }
        const data = await response.json();
        resultPre.textContent = JSON.stringify(data, null, 2);
    } catch (e) {
        errorDiv.textContent = 'Ошибка сети';
    }
});