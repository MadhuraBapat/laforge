import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { CongratsOrganizerComponent } from './congrats-organizer.component';

describe('CongratsOrganizerComponent', () => {
  let component: CongratsOrganizerComponent;
  let fixture: ComponentFixture<CongratsOrganizerComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ CongratsOrganizerComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(CongratsOrganizerComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
