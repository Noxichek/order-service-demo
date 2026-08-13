import { Component, signal } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { OrderDetail } from './components/order-detail/order-detail';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, OrderDetail],
  templateUrl: './app.html',
  styleUrl: './app.scss',
})
export class App {
  protected readonly title = signal('frontend');
}
