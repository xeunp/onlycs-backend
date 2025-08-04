// State Management
const state = {
    items: [],
    filteredItems: [],
    filters: {
        minProfit: 10,
        category: '',
        quality: '',
        minPrice: '',
        maxPrice: '',
        statTrak: false,
        hasStickers: false
    },
    sort: 'profit_desc',
    exchangeRate: 0,
    lastUpdated: null
};

// DOM Elements
const elements = {
    exchangeRate: document.getElementById('exchange-rate'),
    lastUpdated: document.getElementById('last-updated'),
    refreshBtn: document.getElementById('refresh-btn'),
    itemsGrid: document.getElementById('items-grid'),
    itemsCount: document.getElementById('items-count'),
    noItems: document.getElementById('no-items'),

    // Filters
    profitSlider: document.getElementById('profit-slider'),
    profitValue: document.getElementById('profit-value'),
    categoryFilter: document.getElementById('category-filter'),
    qualityFilter: document.getElementById('quality-filter'),
    minPriceFilter: document.getElementById('min-price'),
    maxPriceFilter: document.getElementById('max-price'),
    statTrakFilter: document.getElementById('stattrak-filter'),
    stickersFilter: document.getElementById('stickers-filter'),
    clearFiltersBtn: document.getElementById('clear-filters'),
    resetFiltersBtn: document.getElementById('reset-filters'),

    // Sort
    sortBy: document.getElementById('sort-by')
};

// API Endpoints
const API = {
    exchangeRate: '/api/exchange-rate',
    arbitrage: '/api/arbitrage',
    refresh: '/api/refresh'
};

// Fetch Exchange Rate
async function fetchExchangeRate() {
    try {
        const response = await fetch(API.exchangeRate);
        const data = await response.json();

        state.exchangeRate = data.usdt_to_irr;
        state.lastUpdated = new Date(data.updated_at);

        // Update UI
        elements.exchangeRate.textContent = formatNumber(state.exchangeRate);
        elements.lastUpdated.textContent = formatDate(state.lastUpdated);
    } catch (error) {
        console.error('Error fetching exchange rate:', error);
        elements.exchangeRate.textContent = 'Error';
    }
}

// Fetch Arbitrage Opportunities
async function fetchArbitrageItems() {
    try {
        const minProfit = state.filters.minProfit;
        const response = await fetch(`${API.arbitrage}?min_profit=${minProfit}`);
        const data = await response.json();

        state.items = data.opportunities || [];
        applyFiltersAndSort();
    } catch (error) {
        console.error('Error fetching arbitrage items:', error);
        showError('Failed to load items. Please try again later.');
    }
}

// Refresh Data
async function refreshData() {
    try {
        elements.refreshBtn.disabled = true;
        elements.refreshBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Refreshing...';

        const response = await fetch(API.refresh);
        if (!response.ok) {
            throw new Error('Refresh failed');
        }

        await fetchExchangeRate();
        await fetchArbitrageItems();

        showNotification('Data refreshed successfully!', 'success');
    } catch (error) {
        console.error('Error refreshing data:', error);
        showNotification('Failed to refresh data. Please try again.', 'error');
    } finally {
        elements.refreshBtn.disabled = false;
        elements.refreshBtn.innerHTML = '<i class="fas fa-sync-alt"></i> Refresh Data';
    }
}

// Apply Filters and Sort
function applyFiltersAndSort() {
    // Apply filters
    state.filteredItems = state.items.filter(item => {
        // Min profit percentage
        if (item.profit_percent < state.filters.minProfit) {
            return false;
        }

        // Category filter
        if (state.filters.category && item.category !== state.filters.category) {
            return false;
        }

        // Quality filter
        if (state.filters.quality && item.quality !== state.filters.quality) {
            return false;
        }

        // Price range filters
        if (state.filters.minPrice && item.buy_price_usd < parseFloat(state.filters.minPrice)) {
            return false;
        }

        if (state.filters.maxPrice && item.buy_price_usd > parseFloat(state.filters.maxPrice)) {
            return false;
        }

        // StatTrak filter
        if (state.filters.statTrak && !item.market_hash_name.includes('StatTrak™')) {
            return false;
        }

        // Has stickers filter
        if (state.filters.hasStickers && (!item.stickers || item.stickers.length === 0)) {
            return false;
        }

        return true;
    });

    // Apply sorting
    sortItems();

    // Render items
    renderItems();
}

// Sort Items
function sortItems() {
    state.filteredItems.sort((a, b) => {
        switch (state.sort) {
            case 'profit_desc':
                return b.profit_percent - a.profit_percent;
            case 'profit_asc':
                return a.profit_percent - b.profit_percent;
            case 'price_desc':
                return b.buy_price_usd - a.buy_price_usd;
            case 'price_asc':
                return a.buy_price_usd - b.buy_price_usd;
            case 'name_asc':
                return a.market_hash_name.localeCompare(b.market_hash_name);
            case 'name_desc':
                return b.market_hash_name.localeCompare(a.market_hash_name);
            default:
                return 0;
        }
    });
}

// Render Items
function renderItems() {
    elements.itemsCount.textContent = state.filteredItems.length;

    // Clear items grid
    elements.itemsGrid.innerHTML = '';

    // Show/hide no items message
    if (state.filteredItems.length === 0) {
        elements.noItems.style.display = 'flex';
        return;
    } else {
        elements.noItems.style.display = 'none';
    }

    // Get item template
    const template = document.getElementById('item-template');

    // Create item cards
    state.filteredItems.forEach(item => {
        const card = template.content.cloneNode(true);

        // Set category and quality
        card.querySelector('.item-category').textContent = item.category || 'Unknown';
        card.querySelector('.item-quality').textContent = item.quality || 'Unknown';

        // Set image
        const img = card.querySelector('.item-image img');
        img.src = item.icon_url || '/static/img/placeholder.png';
        img.alt = item.market_hash_name;

        // Set name
        card.querySelector('.item-name').textContent = item.market_hash_name;

        // Set float
        card.querySelector('.item-float').textContent = `Float: ${item.float.toFixed(6)}`;

        // Set prices
        card.querySelector('.buy-price .price-value').textContent = `$${item.buy_price_usd.toFixed(2)}`;
        card.querySelector('.sell-price .price-value').textContent = `$${item.sell_price_usd.toFixed(2)}`;

        // Set profit
        card.querySelector('.profit-amount').textContent = `+$${item.profit_usd.toFixed(2)}`;
        card.querySelector('.profit-percentage').textContent = `+${item.profit_percent.toFixed(2)}%`;

        // Set marketplace
        card.querySelector('.item-marketplace').textContent = item.marketplace;

        // Add stickers if available
        const stickersContainer = card.querySelector('.item-stickers');
        if (item.stickers && item.stickers.length > 0) {
            item.stickers.forEach(stickerUrl => {
                if (stickerUrl) {
                    const sticker = document.createElement('img');
                    sticker.className = 'sticker';
                    sticker.src = stickerUrl;
                    sticker.alt = 'Sticker';
                    stickersContainer.appendChild(sticker);
                }
            });
        } else {
            stickersContainer.style.display = 'none';
        }

        // Add the card to the grid
        elements.itemsGrid.appendChild(card);
    });
}

// Event Listeners
function setupEventListeners() {
    // Profit slider
    elements.profitSlider.addEventListener('input', (e) => {
        const value = e.target.value;
        elements.profitValue.textContent = value;
        state.filters.minProfit = parseInt(value);
        applyFiltersAndSort();
    });

    // Category filter
    elements.categoryFilter.addEventListener('change', (e) => {
        state.filters.category = e.target.value;
        applyFiltersAndSort();
    });

    // Quality filter
    elements.qualityFilter.addEventListener('change', (e) => {
        state.filters.quality = e.target.value;
        applyFiltersAndSort();
    });

    // Min price filter
    elements.minPriceFilter.addEventListener('change', (e) => {
        state.filters.minPrice = e.target.value;
        applyFiltersAndSort();
    });

    // Max price filter
    elements.maxPriceFilter.addEventListener('change', (e) => {
        state.filters.maxPrice = e.target.value;
        applyFiltersAndSort();
    });

    // StatTrak filter
    elements.statTrakFilter.addEventListener('change', (e) => {
        state.filters.statTrak = e.target.checked;
        applyFiltersAndSort();
    });

    // Stickers filter
    elements.stickersFilter.addEventListener('change', (e) => {
        state.filters.hasStickers = e.target.checked;
        applyFiltersAndSort();
    });

    // Clear filters
    elements.clearFiltersBtn.addEventListener('click', clearFilters);
    elements.resetFiltersBtn.addEventListener('click', clearFilters);

    // Sort selector
    elements.sortBy.addEventListener('change', (e) => {
        state.sort = e.target.value;
        applyFiltersAndSort();
    });

    // Refresh button
    elements.refreshBtn.addEventListener('click', refreshData);
}

// Clear all filters
function clearFilters() {
    // Reset filter values
    state.filters = {
        minProfit: 10,
        category: '',
        quality: '',
        minPrice: '',
        maxPrice: '',
        statTrak: false,
        hasStickers: false
    };

    // Reset UI elements
    elements.profitSlider.value = 10;
    elements.profitValue.textContent = '10';
    elements.categoryFilter.value = '';
    elements.qualityFilter.value = '';
    elements.minPriceFilter.value = '';
    elements.maxPriceFilter.value = '';
    elements.statTrakFilter.checked = false;
    elements.stickersFilter.checked = false;

    // Apply updated filters
    applyFiltersAndSort();
}

// Show notification
function showNotification(message, type = 'info') {
    // Implementation for showing toast notifications
    // This is a simple implementation, you might want to use a library
    const notification = document.createElement('div');
    notification.className = `notification ${type}`;
    notification.textContent = message;

    document.body.appendChild(notification);

    // Animate in
    setTimeout(() => {
        notification.classList.add('show');
    }, 10);

    // Remove after 3 seconds
    setTimeout(() => {
        notification.classList.remove('show');
        setTimeout(() => {
            notification.remove();
        }, 300);
    }, 3000);
}

// Show error in the items grid
function showError(message) {
    elements.itemsGrid.innerHTML = `
        <div class="error-message">
            <i class="fas fa-exclamation-circle"></i>
            <p>${message}</p>
        </div>
    `;
}

// Helper Functions
function formatNumber(num) {
    return new Intl.NumberFormat().format(num);
}

function formatDate(date) {
    if (!date) return '-';
    return new Intl.DateTimeFormat('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    }).format(date);
}

// Initialize the application
async function init() {
    setupEventListeners();

    // Show loading state
    elements.itemsGrid.innerHTML = `
        <div class="loading">
            <div class="spinner"></div>
            <p>Loading items...</p>
        </div>
    `;

    // Fetch initial data
    await fetchExchangeRate();
    await fetchArbitrageItems();
}

// Start the app
document.addEventListener('DOMContentLoaded', init);