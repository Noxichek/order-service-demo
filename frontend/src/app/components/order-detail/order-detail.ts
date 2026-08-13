import { HttpClient } from '@angular/common/http';
import { Component, signal, WritableSignal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-order-detail',
  imports: [CommonModule, FormsModule],
  templateUrl: './order-detail.html',
  styleUrl: './order-detail.scss',
})
export class OrderDetail {
  public orderId: string = '';
  public order$$: WritableSignal<any> = signal(null);
  public error: string = '';

  constructor(private http: HttpClient) {}

  fetchOrder() {
    if (!this.orderId.trim()) return;
    this.error = '';
    this.order$$.set(null);
    this.http.get(`http://localhost:8081/order/${this.orderId}`).subscribe({
      next: (data) => {
        console.log(data);
        this.order$$.set(data);
      },
      error: (err) => {
        this.error = err.status === 404 ? 'Заказ не найден' : 'Ошибка сервера';
      },
    });
  }
}
